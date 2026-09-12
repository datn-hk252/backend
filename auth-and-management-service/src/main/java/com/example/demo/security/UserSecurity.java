package com.example.demo.security;

import com.example.demo.repository.UserRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

/**
 * Answers "is this the account you are signed in as?" for {@code @PreAuthorize}.
 *
 * The JWT filter puts the email on the security context while the endpoints are
 * addressed by id, so the two have to be reconciled somewhere. Doing it here
 * keeps the rule on the method it guards, where a reader looking at an endpoint
 * can see who may call it.
 */
@Component("userSecurity")
@RequiredArgsConstructor
public class UserSecurity {

    private final UserRepository userRepository;

    /** True when {@code id} names the account belonging to {@code email}. */
    public boolean isSelf(Long id, String email) {
        if (id == null || email == null || email.isBlank()) {
            return false;
        }
        return userRepository.findById(id)
                .map(user -> email.equalsIgnoreCase(user.getEmail()))
                .orElse(false);
    }
}
