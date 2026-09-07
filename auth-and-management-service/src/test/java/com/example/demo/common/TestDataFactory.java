package com.example.demo.common;

import com.example.demo.dto.auth.LoginRequest;
import com.example.demo.dto.auth.RegisterRequest;
import com.example.demo.enums.*;
import com.example.demo.model.*;

import java.time.LocalDateTime;

public final class TestDataFactory {

    private TestDataFactory() {}

    public static User adminUser() {
        return User.builder()
                .id(1L)
                .name("Admin User")
                .email("admin@test.com")
                .password("$2a$10$encoded_password")
                .role(UserRole.ROLE_ADMIN)
                .code("ADMIN001")
                .active(true)
                .build();
    }

    public static User managerUser() {
        return User.builder()
                .id(2L)
                .name("Manager User")
                .email("manager@test.com")
                .password("$2a$10$encoded_password")
                .role(UserRole.ROLE_MANAGER)
                .code("MGR001")
                .active(true)
                .build();
    }

    public static User regularUser() {
        return User.builder()
                .id(3L)
                .name("Regular User")
                .email("user@test.com")
                .password("$2a$10$encoded_password")
                .role(UserRole.ROLE_USER)
                .code("USR001")
                .active(true)
                .build();
    }

    public static User userWithId(Long id) {
        return regularUser().toBuilder().id(id)
                .email("user" + id + "@test.com")
                .code("USR" + id)
                .build();
    }

    public static LoginRequest loginRequest(String email, String password) {
        return LoginRequest.builder().email(email).password(password).build();
    }

    public static RegisterRequest registerRequest(String email, String code) {
        return RegisterRequest.builder()
                .name("New Member")
                .email(email)
                .role(UserRole.ROLE_USER)
                .code(code)
                .build();
    }

    public static PasswordResetToken validToken(User user) {
        return PasswordResetToken.builder()
                .id(1L)
                .token("valid-reset-token-uuid")
                .user(user)
                .expiryDate(LocalDateTime.now().plusMinutes(10))
                .createdAt(LocalDateTime.now())
                .used(false)
                .build();
    }

    public static PasswordResetToken expiredToken(User user) {
        return PasswordResetToken.builder()
                .id(2L)
                .token("expired-token")
                .user(user)
                .expiryDate(LocalDateTime.now().minusMinutes(5))
                .createdAt(LocalDateTime.now().minusMinutes(20))
                .used(false)
                .build();
    }

    public static PasswordResetToken usedToken(User user) {
        return PasswordResetToken.builder()
                .id(3L)
                .token("used-token")
                .user(user)
                .expiryDate(LocalDateTime.now().plusMinutes(10))
                .createdAt(LocalDateTime.now())
                .used(true)
                .build();
    }
}