package com.example.demo.config;

import com.example.demo.enums.UserRole;
import com.example.demo.model.LmsRoleMapping;
import com.example.demo.model.Role;
import com.example.demo.model.User;
import com.example.demo.repository.LmsRoleMappingRepository;
import com.example.demo.repository.RoleRepository;
import com.example.demo.repository.UserRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.annotation.Order;
import org.springframework.security.crypto.password.PasswordEncoder;

@Slf4j
@Configuration
@RequiredArgsConstructor
@Order(1)
public class DataInitializer implements CommandLineRunner {

    private final UserRepository userRepository;
    private final RoleRepository roleRepository;
    private final LmsRoleMappingRepository lmsMappingRepository;
    private final PasswordEncoder passwordEncoder;
    private final org.springframework.jdbc.core.JdbcTemplate jdbcTemplate;
    private final com.example.demo.service.user.UserSyncService userSyncService;

    @org.springframework.beans.factory.annotation.Value("${app.admin.password:hehehe}")
    private String adminPassword;

    @org.springframework.beans.factory.annotation.Value("${app.admin.email:phucnhan289@gmail.com}")
    private String adminEmail;

    @Override
    public void run(String... args) {
        try {
            log.info("Dropping database check constraints for dynamic fields...");
            jdbcTemplate.execute("ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check");
            jdbcTemplate.execute("ALTER TABLE users DROP CONSTRAINT IF EXISTS users_type_check");
            jdbcTemplate.execute("ALTER TABLE users DROP CONSTRAINT IF EXISTS users_team_check");
            log.info("Successfully dropped check constraints.");
        } catch (Exception e) {
            log.error("Failed to drop check constraints: {}", e.getMessage());
        }
        seedRoles();
        seedAdminUser();

        // The admin seeded above never goes through a path that syncs it to LMS, so on a
        // fresh database it can log in but has no role there. Runs asynchronously and
        // waits for lms-service on its own thread, so startup is not held up.
        log.info("Triggering automatic startup sync of existing users to lms-service...");
        try {
            userSyncService.syncUsersToLms(userRepository.findAll());
        } catch (Exception e) {
            log.error("Startup user sync to LMS failed: {}", e.getMessage());
        }
    }

    /**
     * Seed the dynamic roles table and default LMS mappings.
     * Idempotent - skips if roles already exist.
     */
    private void seedRoles() {
        seedRole(UserRole.ROLE_ADMIN, "Quản trị viên", "ADMIN", "Administrator");
        seedRole(UserRole.ROLE_MANAGER, "Giáo viên", "TEACHER", "Manager");
        seedRole(UserRole.ROLE_USER, "Học viên", "STUDENT", "Member");
        log.debug("Role seeding complete");
    }

    /**
     * Create the role if it is missing, otherwise leave it alone - except for the
     * one case of a role still carrying the club-era display name it was seeded
     * with. Seeding runs once, so renaming the constants above would otherwise
     * never reach a database created before this change; renaming only that exact
     * literal cannot clobber a name an admin chose deliberately.
     */
    private void seedRole(String roleName, String displayName, String defaultLmsRole,
                          String legacyDisplayName) {
        var existing = roleRepository.findByName(roleName);
        if (existing.isPresent()) {
            var role = existing.get();
            if (legacyDisplayName.equals(role.getDisplayName())) {
                role.setDisplayName(displayName);
                roleRepository.save(role);
                log.info("Renamed role {} from \"{}\" to \"{}\"", roleName, legacyDisplayName, displayName);
            }
            return;
        }

        var role = roleRepository.save(Role.builder()
                .name(roleName)
                .displayName(displayName)
                .isSystem(true)
                .build());

        lmsMappingRepository.save(LmsRoleMapping.builder()
                .authRole(role)
                .lmsRole(defaultLmsRole)
                .build());

        log.info("Seeded role {} -> LMS [{}]", roleName, defaultLmsRole);
    }

    private void seedAdminUser() {
        if (userRepository.count() > 0) {
            log.debug("Database already seeded, skipping admin user creation");
            return;
        }

        var admin = User.builder()
                .name("Quản trị viên")
                .email(adminEmail)
                .password(passwordEncoder.encode(adminPassword))
                .role(UserRole.ROLE_ADMIN)
                .code("000000")
                .active(true)
                .build();

        userRepository.save(admin);
        log.info("Default admin user created: {}", adminEmail);
    }
}