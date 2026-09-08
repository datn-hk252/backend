package com.example.demo.dto.user;

import com.example.demo.enums.AuthProvider;

import com.example.demo.model.User;
import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class UserResponse {

    private Long       id;
    private String     name;
    private String     email;
    private String     role;
    private java.util.List<String> roles;
    private java.util.List<String> lmsRoles;
    private String       code;
    private Boolean      active;
    private AuthProvider authProvider;
    private Boolean      pendingApproval;
    private String     profilePicture;

    public static UserResponse fromEntity(User user) {
        return UserResponse.builder()
                .id(user.getId())
                .name(user.getName())
                .email(user.getEmail())
                .role(user.getRole())
                .roles(new java.util.ArrayList<>(user.effectiveRoles()))
                .lmsRoles(user.getLmsRoles() == null ? java.util.List.of() : new java.util.ArrayList<>(user.getLmsRoles()))
                .code(user.getCode())
                .active(user.getActive())
                .profilePicture(user.getProfilePicture())
                .authProvider(user.getAuthProvider())
                .pendingApproval(user.getPendingApproval())
                .build();
    }
}
