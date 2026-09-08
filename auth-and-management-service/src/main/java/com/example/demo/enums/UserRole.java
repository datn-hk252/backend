package com.example.demo.enums;

/**
 * Well-known role constants.
 *
 * Converted from enum to String constants so that roles are fully dynamic
 * (stored in the {@code roles} DB table) while keeping backward compatibility
 * with every call-site that references {@code UserRole.ROLE_ADMIN} etc.
 */
public final class UserRole {
    public static final String ROLE_ADMIN   = "ROLE_ADMIN";
    public static final String ROLE_TEACHER = "ROLE_TEACHER";
    public static final String ROLE_STUDENT = "ROLE_STUDENT";

    /** Club-era names these two replaced; DataInitializer renames what is stored. */
    public static final String LEGACY_ROLE_MANAGER = "ROLE_MANAGER";
    public static final String LEGACY_ROLE_USER    = "ROLE_USER";

    private UserRole() {}
}
