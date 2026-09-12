package com.example.demo.controller;

import com.example.demo.dto.auth.MessageResponse;
import com.example.demo.dto.user.ChangePasswordRequest;
import com.example.demo.dto.user.UpdateUserRequest;
import com.example.demo.dto.user.UpdateUserRoleRequest;
import com.example.demo.dto.user.UserResponse;
import com.example.demo.dto.common.PageResponse;
import com.example.demo.service.user.UserService;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.util.List;
import java.util.Map;

/**
 * UserController - REST API cho user management.
 *
 * Fixes:
 *   - public class (không còn package-private).
 *   - Trả UserResponse thay vì User entity - không lộ internal model.
 *   - updateUser nhận UpdateUserRequest thay vì User entity.
 *     Tránh mass assignment: client không thể tự set role, totalScore qua endpoint này.
 */
@RestController
@RequestMapping("/api/users")
@RequiredArgsConstructor
public class UserController {

    private final UserService userService;

    /**
     * The whole directory: every name, email, code and role.
     *
     * Only the admin screens ask for it - the user table and the people pickers
     * on the class screen - and it is personal data about everyone at the
     * centre, so a learner holding a valid token had no business reading it.
     */
    @PreAuthorize("hasRole('ADMIN')")
    @GetMapping
    public ResponseEntity<PageResponse<UserResponse>> getAll(
            @RequestParam(defaultValue = "0") int page,
            @RequestParam(name = "page_size", defaultValue = "50") int pageSize,
            @RequestParam(defaultValue = "") String query,
            @RequestParam(defaultValue = "") String role,
            @RequestParam(name = "sort_by", defaultValue = "id") String sortBy,
            @RequestParam(name = "sort_dir", defaultValue = "desc") String sortDirection) {
        return ResponseEntity.ok(userService.getUsers(
                page, pageSize, query, role, sortBy, sortDirection));
    }

    @PreAuthorize("hasRole('ADMIN') or @userSecurity.isSelf(#id, authentication.name)")
    @GetMapping("/{id}")
    public ResponseEntity<UserResponse> getById(@PathVariable Long id) {
        return ResponseEntity.ok(userService.getUserById(id));
    }

    @PreAuthorize("hasRole('ADMIN') or @userSecurity.isSelf(#id, authentication.name)")
    @PutMapping("/{id}")
    public ResponseEntity<UserResponse> update(
            @PathVariable Long id,
            @Valid @RequestBody UpdateUserRequest req) {
        return ResponseEntity.ok(userService.updateUser(id, req));
    }

    @PreAuthorize("hasRole('ADMIN')")
    @PatchMapping("/{id}/role")
    public ResponseEntity<UserResponse> updateRole(
            @PathVariable Long id,
            @Valid @RequestBody UpdateUserRoleRequest req) {
        return ResponseEntity.ok(userService.updateRole(id, req.getRole()));
    }

    @PreAuthorize("hasRole('ADMIN') or @userSecurity.isSelf(#id, authentication.name)")
    @PostMapping("/{id}/change-password")
    public ResponseEntity<Map<String, String>> changePassword(
            @PathVariable Long id,
            @Valid @RequestBody ChangePasswordRequest req) {
        userService.changePassword(id, req.getCurrentPassword(), req.getNewPassword());
        return ResponseEntity.ok(Map.of("message", "Password changed successfully"));
    }

    @PreAuthorize("hasRole('ADMIN') or @userSecurity.isSelf(#id, authentication.name)")
    @PostMapping("/{id}/upload-picture")
    public ResponseEntity<Map<String, String>> uploadPicture(
            @PathVariable Long id,
            @RequestParam("file") MultipartFile file) {
        String url = userService.uploadProfilePicture(id, file);
        return ResponseEntity.ok(Map.of("profilePicture", url));
    }

    @PreAuthorize("hasRole('ADMIN')")
    @PatchMapping("/{id}/status")
    public ResponseEntity<UserResponse> toggleStatus(@PathVariable Long id) {
        return ResponseEntity.ok(userService.toggleActive(id));
    }

    @PreAuthorize("hasRole('ADMIN')")
    @DeleteMapping("/{id}")
    public ResponseEntity<Void> delete(@PathVariable Long id) {
        userService.deleteUser(id);
        return ResponseEntity.noContent().build();
    }

    /**
     * Issues a fresh temporary password and mails it.
     *
     * The password from an import exists only in the message that carried it -
     * it is stored hashed, so a welcome mail that failed leaves an account
     * nobody can sign in to and nothing to look up. This is the way back.
     */
    @PreAuthorize("hasRole('ADMIN')")
    @PostMapping("/{id}/resend-password")
    public ResponseEntity<MessageResponse> resendPassword(@PathVariable Long id) {
        userService.resendTemporaryPassword(id);
        return ResponseEntity.ok(new MessageResponse("Đã gửi lại mật khẩu tạm thời"));
    }

    @PreAuthorize("hasRole('ADMIN')")
    @GetMapping("/pending")
    public ResponseEntity<List<UserResponse>> getPendingUsers() {
        return ResponseEntity.ok(userService.getPendingUsers());
    }

    @PreAuthorize("hasRole('ADMIN')")
    @PatchMapping("/{id}/approve")
    public ResponseEntity<UserResponse> approveUser(@PathVariable Long id) {
        return ResponseEntity.ok(userService.approveUser(id));
    }

    @PreAuthorize("hasRole('ADMIN')")
    @PatchMapping("/{id}/reject")
    public ResponseEntity<UserResponse> rejectUser(@PathVariable Long id) {
        return ResponseEntity.ok(userService.rejectUser(id));
    }
}
