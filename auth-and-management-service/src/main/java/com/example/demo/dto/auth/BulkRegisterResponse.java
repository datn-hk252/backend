package com.example.demo.dto.auth;

import com.example.demo.dto.user.UserResponse;
import lombok.AllArgsConstructor;
import lombok.Data;

import java.util.List;

/**
 * Result of an import.
 *
 * The accounts exist whether or not their welcome mail arrived, and the
 * password only ever existed in that mail - it is stored hashed, so nobody can
 * look it up afterwards. An address that failed therefore leaves an account
 * nobody can sign in to, and the only person able to notice is the admin who
 * ran the import. Hence this second list.
 */
@Data
@AllArgsConstructor
public class BulkRegisterResponse {

    private List<UserResponse> users;

    /** Addresses whose welcome mail did not go out. Empty on a clean import. */
    private List<String> emailFailures;

    /**
     * True when the mail was still being sent as the response went out, so
     * {@code emailFailures} is what was known at that moment rather than the
     * final answer.
     */
    private boolean emailPending;
}
