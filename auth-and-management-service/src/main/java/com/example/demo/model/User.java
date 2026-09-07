package com.example.demo.model;

import com.example.demo.enums.AuthProvider;

import jakarta.persistence.*;
import lombok.*;
import java.util.LinkedHashSet;
import java.util.Set;
import com.fasterxml.jackson.annotation.JsonIgnore;

@Entity
@Table(name = "users")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder(toBuilder = true)
public class User {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false, length = 100)
    private String name;

    @Column(nullable = false, unique = true, length = 100)
    private String email;

    @Column(nullable = false)
    @JsonIgnore
    private String password;

    @Column(nullable = false, length = 50)
    private String role;

    @ElementCollection(fetch = FetchType.EAGER)
    @CollectionTable(name = "user_roles", joinColumns = @JoinColumn(name = "user_id"),
            uniqueConstraints = @UniqueConstraint(columnNames = {"user_id", "role_name"}))
    @Column(name = "role_name", nullable = false, length = 50)
    @Builder.Default
    private Set<String> roles = new LinkedHashSet<>();

    @ElementCollection(fetch = FetchType.EAGER)
    @CollectionTable(name = "user_lms_roles", joinColumns = @JoinColumn(name = "user_id"),
            uniqueConstraints = @UniqueConstraint(columnNames = {"user_id", "lms_role"}))
    @Column(name = "lms_role", nullable = false, length = 20)
    @Builder.Default
    private Set<String> lmsRoles = new LinkedHashSet<>();

    /** Student or teacher number. Printed on rosters, so it stays unique. */
    @Column(nullable = false, unique = true, length = 100)
    private String code;

    /**
     * Club-era columns kept only because the database says NOT NULL.
     *
     * <p>The fork divided members into a team (Research, Engineer, ...) and a
     * training type (CLC, TN, DT); an English centre has neither. Hibernate runs
     * with {@code ddl-auto=update}, which never drops a constraint, so on any
     * database created before this change the columns must still receive a
     * value. Nothing reads them.
     */
    public static final String LEGACY_UNUSED = "N/A";

    @Column(nullable = false, length = 50)
    @Builder.Default
    private String team = LEGACY_UNUSED;

    @Column(nullable = false, length = 20)
    @Builder.Default
    private String type = LEGACY_UNUSED;

    @Column(nullable = false)
    @Builder.Default
    private Boolean active = true;

    // Path to profile picture
    @Column(nullable = true)
    private String profilePicture;

    @Column(nullable = false)
    @Builder.Default
    private Integer totalScore = 0;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 10)
    @Builder.Default
    private AuthProvider authProvider = AuthProvider.LOCAL;

    @Column(unique = true)
    private String googleId;

    @Column(nullable = false)
    @Builder.Default
    private Boolean pendingApproval = false;

    public Set<String> effectiveRoles() {
        LinkedHashSet<String> result = new LinkedHashSet<>();
        if (role != null && !role.isBlank()) result.add(role);
        if (roles != null) result.addAll(roles);
        return result;
    }
}
