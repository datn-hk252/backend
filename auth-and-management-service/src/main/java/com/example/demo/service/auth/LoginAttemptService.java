package com.example.demo.service.auth;

import com.example.demo.exception.TooManyRequestsException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.Locale;

/**
 * NFR-SEC-05: stop an automated password guesser without handing anyone a way
 * to lock a colleague out.
 *
 * <p>Counting failures per account alone would do exactly that: knowing
 * someone's email would be enough to keep them out for fifteen minutes at a
 * time. Counting per address alone is no good either - a guesser just rotates
 * addresses. So the lockout is keyed on the <em>pair</em>: five wrong passwords
 * for one account from one address stops that attempt, while the real owner
 * signing in from anywhere else is unaffected.
 *
 * <p>A second, looser counter watches the address on its own, to catch the
 * other shape of attack - one machine trying one common password against many
 * accounts, which never trips any single pair.
 *
 * <p>Both counters expire on their own through the Redis TTL, so nothing has to
 * sweep them. If Redis is unreachable the checks pass: an outage should not
 * lock the whole centre out of the platform.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class LoginAttemptService {

    private static final String PAIR_KEY = "auth:login-fail:%s:%s";
    private static final String ADDRESS_KEY = "auth:login-fail:ip:%s";

    private final StringRedisTemplate redis;

    @Value("${security.login-lockout.max-attempts:5}")
    private int maxAttempts;

    @Value("${security.login-lockout.lockout-minutes:15}")
    private int lockoutMinutes;

    /**
     * How many failures one address may accumulate across all accounts before
     * it is shut out. A person mistyping their own password a few times stays
     * well under it; a script working through a list of emails does not.
     */
    @Value("${security.login-lockout.max-attempts-per-ip:20}")
    private int maxAttemptsPerAddress;

    /**
     * Call before touching the database, so a locked-out guesser costs nothing
     * but a Redis read.
     */
    public void assertNotLocked(String email, String ip) {
        if (count(pairKey(email, ip)) >= maxAttempts || count(addressKey(ip)) >= maxAttemptsPerAddress) {
            throw new TooManyRequestsException(
                    "Bạn đã đăng nhập sai quá nhiều lần. Vui lòng thử lại sau "
                    + lockoutMinutes + " phút.");
        }
    }

    /** Record a wrong password and start the window if this was the first one. */
    public void recordFailure(String email, String ip) {
        increment(pairKey(email, ip));
        increment(addressKey(ip));
    }

    /** A correct password clears the slate for that pair. */
    public void recordSuccess(String email, String ip) {
        try {
            redis.delete(pairKey(email, ip));
        } catch (Exception e) {
            log.warn("Could not clear login failures for {}: {}", email, e.getMessage());
        }
    }

    // ── Helpers ───────────────────────────────────────────────────────────────

    private long count(String key) {
        try {
            String value = redis.opsForValue().get(key);
            return value == null ? 0 : Long.parseLong(value);
        } catch (Exception e) {
            // Redis down, or a value somebody else wrote. Fail open.
            log.warn("Login lockout check skipped for {}: {}", key, e.getMessage());
            return 0;
        }
    }

    private void increment(String key) {
        try {
            Long count = redis.opsForValue().increment(key);
            // Start the clock on the first failure only, so the window is a
            // fixed fifteen minutes from that point rather than fifteen minutes
            // after whichever attempt happened to be last.
            if (count != null && count == 1L) {
                redis.expire(key, Duration.ofMinutes(lockoutMinutes));
            }
        } catch (Exception e) {
            log.warn("Could not record a login failure for {}: {}", key, e.getMessage());
        }
    }

    private String pairKey(String email, String ip) {
        return String.format(PAIR_KEY, normalise(email), ip);
    }

    private String addressKey(String ip) {
        return String.format(ADDRESS_KEY, ip);
    }

    private String normalise(String email) {
        return email == null ? "" : email.trim().toLowerCase(Locale.ROOT);
    }
}
