package com.example.demo.dto.auth;


import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@AllArgsConstructor
@NoArgsConstructor
@Builder
public class RegisterRequest {
    private String name;
    private String email;
    private String role;
    /** Multiple auth roles. The legacy role field remains the primary role. */
    private java.util.List<String> roles;
    /** Optional LMS roles independent from auth roles: ADMIN, TEACHER, STUDENT. */
    private java.util.List<String> lmsRoles;
    private String code;
}
