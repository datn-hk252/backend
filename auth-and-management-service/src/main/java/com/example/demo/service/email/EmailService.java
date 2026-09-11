package com.example.demo.service.email;

import java.util.Map;
import java.util.List;
import java.util.concurrent.CompletableFuture;

public interface EmailService {
    void sendWelcomeEmail(String to, String name, String tempPassword);

    void sendPasswordChangeConfirmation(String to, String name, String token);

    void sendPasswordChangedNotification(String to, String name);

    void sendForgotPasswordEmail(String to, String name, String token);

    void sendRecruitmentConfirmationEmail(String to, String name, String department);

    CompletableFuture<Void> sendWelcomeEmailAsync(String to, String name, String tempPassword);

    /**
     * Sends every welcome mail in the batch and reports which addresses failed.
     *
     * One bad address must not stop the rest, so failures are collected rather
     * than thrown - but they are returned, because an account whose password
     * never arrived cannot be signed in to and the password is not recoverable.
     */
    CompletableFuture<java.util.List<String>> sendWelcomeBatch(Map<String, String> emailToPassword, Map<String, String> emailToName);

    CompletableFuture<Void> sendPasswordChangeConfirmationAsync(String to, String name, String token);

    CompletableFuture<Void> sendPasswordChangedNotificationAsync(String to, String name);

    CompletableFuture<Void> sendForgotPasswordEmailAsync(String to, String name, String token);

    CompletableFuture<Void> sendRecruitmentConfirmationAsync(String to, String name, String department);

    CompletableFuture<Void> sendAdminMailAsync(String to, List<String> cc, List<String> bcc, String subject, String body, String signatureType, String templateType);
}
