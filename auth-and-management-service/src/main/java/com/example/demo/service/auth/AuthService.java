package com.example.demo.service.auth;

import com.example.demo.dto.auth.BulkRegisterRequest;
import com.example.demo.dto.auth.LoginRequest;
import com.example.demo.dto.auth.RegisterRequest;
import com.example.demo.exception.BadRequestException;
import com.example.demo.model.User;
import com.example.demo.repository.UserRepository;
import com.example.demo.repository.RoleRepository;
import com.example.demo.service.email.EmailService;
import com.example.demo.service.user.UserSyncService;
import com.example.demo.strategy.RoleResolutionStrategy;
import com.example.demo.utils.PasswordGenerator;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;
import java.util.Locale;
import java.util.stream.Collectors;

@Slf4j
@Service
@RequiredArgsConstructor
public class AuthService {

    private final UserRepository userRepository;
    private final RoleRepository roleRepository;
    private final JwtService jwtService;
    private final PasswordEncoder passwordEncoder;
    private final EmailService emailService;
    private final UserSyncService userSyncService;
    private final RoleResolutionStrategy roleStrategy;
    private final LoginAttemptService loginAttempts;

    @Value("${app.default-role:ROLE_STUDENT}")
    private String defaultRole;

    /**
     * @param ip where the attempt came from, for the NFR-SEC-05 lockout.
     */
    public User authenticate(LoginRequest request, String ip) {
        // First, so a caller already locked out costs one Redis read and no
        // database work at all.
        loginAttempts.assertNotLocked(request.getEmail(), ip);

        var user = userRepository.findByEmail(request.getEmail()).orElse(null);
        if (user == null || !passwordEncoder.matches(request.getPassword(), user.getPassword())) {
            // One message for both cases: saying which of the two was wrong
            // would turn this endpoint into a way to test whether an email is
            // registered here.
            loginAttempts.recordFailure(request.getEmail(), ip);
            throw new BadRequestException("Invalid email or password");
        }

        if (!user.getActive()) {
            // The password was right, so this is the owner rather than a
            // guesser; their attempts should not count towards a lockout.
            loginAttempts.recordSuccess(request.getEmail(), ip);
            if (user.getPendingApproval()) {
                throw new BadRequestException("Tài khoản của bạn đang chờ admin duyệt. Vui lòng đợi.");
            }
            throw new BadRequestException("Tài khoản của bạn đã bị khóa. Vui lòng liên hệ quản trị viên.");
        }

        loginAttempts.recordSuccess(request.getEmail(), ip);
        return user;
    }

    public String generateToken(User user) {
        var tokenRoles = user.getLmsRoles() != null && !user.getLmsRoles().isEmpty()
                ? user.getLmsRoles().stream().distinct().toList()
                : roleStrategy.resolveAll(user.effectiveRoles());
        return jwtService.generateToken(user.getId(), user.getEmail(),
                                        tokenRoles);
    }

    public String generateRefreshToken(User user) {
        return jwtService.generateRefreshToken(user.getId(), user.getEmail());
    }

    public boolean validateToken(String token) {
        return jwtService.validateToken(token);
    }

    public String extractEmail(String token) {
        return jwtService.extractEmail(token);
    }

    @Transactional
    public List<User> bulkRegister(BulkRegisterRequest request) {
        var registrations = request == null ? null : request.getUsers();
        if (registrations == null || registrations.isEmpty()) {
            throw new BadRequestException("Import batch must contain at least one user");
        }
        if (registrations.size() > 2000) {
            throw new BadRequestException("A single import is limited to 2000 users");
        }

        var existingRoles = roleRepository.findAll().stream()
                .map(role -> role.getName().toUpperCase(Locale.ROOT))
                .collect(Collectors.toSet());
        List<String> errors = new java.util.ArrayList<>();
        List<PreparedRegistration> prepared = new java.util.ArrayList<>();
        Set<String> seenEmails = new java.util.HashSet<>();
        Set<String> seenCodes = new java.util.HashSet<>();

        for (int index = 0; index < registrations.size(); index++) {
            RegisterRequest reg = registrations.get(index);
            int row = index + 2; // spreadsheet header is row 1
            String name = clean(reg.getName());
            String email = clean(reg.getEmail()).toLowerCase(Locale.ROOT);
            String code = clean(reg.getCode());
            if (name.isBlank()) errors.add("Row " + row + ": name is required");
            if (!email.matches("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$")) errors.add("Row " + row + ": invalid email");
            if (code.isBlank()) errors.add("Row " + row + ": code is required");
            if (!seenEmails.add(email)) errors.add("Row " + row + ": duplicate email in file: " + email);
            if (!seenCodes.add(code)) errors.add("Row " + row + ": duplicate code in file: " + code);

            LinkedHashSet<String> roles = new LinkedHashSet<>();
            if (reg.getRoles() != null) reg.getRoles().stream().map(this::normalizeRole).forEach(roles::add);
            if (roles.isEmpty()) roles.add(normalizeRole(reg.getRole()));
            for (String role : roles) {
                if (!existingRoles.contains(role)) errors.add("Row " + row + ": unknown role " + role);
            }

            LinkedHashSet<String> lmsRoles = new LinkedHashSet<>();
            if (reg.getLmsRoles() != null) {
                reg.getLmsRoles().stream()
                        .filter(value -> value != null)
                        .flatMap(value -> java.util.Arrays.stream(value.split("[;,]")))
                        .map(this::normalizeLmsRole)
                        .filter(role -> !role.isBlank())
                        .forEach(lmsRoles::add);
            }
            for (String lmsRole : lmsRoles) {
                if (!Set.of("ADMIN", "TEACHER", "STUDENT").contains(lmsRole)) {
                    errors.add("Row " + row + ": unknown LMS role " + lmsRole);
                }
            }

            prepared.add(new PreparedRegistration(name, email, code, roles, lmsRoles));
        }

        var emails = prepared.stream().map(PreparedRegistration::email).toList();
        var codes = prepared.stream().map(PreparedRegistration::code).toList();
        var duplicateEmails = userRepository.findExistingEmails(emails);
        var duplicateCodes = userRepository.findExistingCodes(codes);
        if (!duplicateEmails.isEmpty()) errors.add("Emails already in database: " + String.join(", ", duplicateEmails));
        if (!duplicateCodes.isEmpty()) errors.add("Codes already in database: " + String.join(", ", duplicateCodes));
        if (!errors.isEmpty()) throw new BadRequestException(String.join("; ", errors));

        Map<String, String> emailToPassword = new java.util.LinkedHashMap<>();
        Map<String, String> emailToName    = new java.util.LinkedHashMap<>();

        List<User> users = prepared.stream()
                .map(item -> {
                    String pwd = PasswordGenerator.generateStrongPassword();
                    emailToPassword.put(item.email(), pwd);
                    emailToName.put(item.email(), item.name());
                    String primaryRole = item.roles().iterator().next();

                    return User.builder()
                            .name(item.name())
                            .email(item.email())
                            .password(passwordEncoder.encode(pwd))
                            .role(primaryRole)
                            .roles(new LinkedHashSet<>(item.roles()))
                            .lmsRoles(new LinkedHashSet<>(item.lmsRoles()))
                            .code(item.code())
                            .active(true)
                            .build();
                })
                .collect(Collectors.toList());

        List<User> saved = userRepository.saveAll(users);
        log.info("Bulk registered {} users", saved.size());

        emailService.sendWelcomeBatch(emailToPassword, emailToName)
                    .exceptionally(ex -> { log.error("Batch email error: {}", ex.getMessage()); return null; });

        userSyncService.syncUsers(saved)
                       .exceptionally(ex -> { log.error("LMS sync error: {}", ex.getMessage()); return null; });

        return saved;
    }

    private String clean(String value) {
        return value == null ? "" : value.trim();
    }

    private String normalizeRole(String role) {
        String normalized = clean(role);
        if (normalized.isBlank()) normalized = defaultRole;
        normalized = normalized.toUpperCase(Locale.ROOT);
        return normalized.startsWith("ROLE_") ? normalized : "ROLE_" + normalized;
    }

    private String normalizeLmsRole(String role) {
        return clean(role).toUpperCase(Locale.ROOT).replaceFirst("^LMS:", "");
    }

    private record PreparedRegistration(
            String name,
            String email,
            String code,
            LinkedHashSet<String> roles,
            LinkedHashSet<String> lmsRoles) {}
}
