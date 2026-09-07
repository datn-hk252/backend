package com.example.demo.dto.auth;

import jakarta.validation.constraints.NotBlank;
import lombok.Data;

@Data
public class GoogleRegisterRequest {
    @NotBlank(message = "Google ID token is required")
    private String idToken;

    @NotBlank(message = "Name is required")
    private String name;

    @NotBlank(message = "Student/member code is required")
    private String code;
}
