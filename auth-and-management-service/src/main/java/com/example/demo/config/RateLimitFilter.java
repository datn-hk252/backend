package com.example.demo.config;

import com.example.demo.service.auth.JwtService;
import com.example.demo.utils.ClientIp;
import com.fasterxml.jackson.databind.ObjectMapper;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.time.Duration;
import java.time.Instant;

/**
 * NFR-SEC-06: cap how fast one account or one address may call the API, so a
 * script cannot walk the whole user list or hammer the login endpoint.
 *
 * <p>Fixed one-minute windows, the same shape lms-service already uses, kept
 * deliberately: two services with two different throttling models would be
 * harder to reason about than one slightly coarse model everywhere. The cost is
 * that a caller can send up to twice the quota across a window boundary; for
 * limits meant to stop bulk downloading rather than to smooth load, that is
 * acceptable.
 *
 * <p>Runs before authentication so an unauthenticated flood is also capped, and
 * falls through when Redis is unreachable rather than locking everybody out.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class RateLimitFilter extends OncePerRequestFilter {

    private static final String ACCOUNT_KEY = "auth:rate:user:%s";
    private static final String ADDRESS_KEY = "auth:rate:ip:%s";
    private static final Duration WINDOW = Duration.ofMinutes(1);

    private final StringRedisTemplate redis;
    private final JwtService jwtService;
    private final ObjectMapper objectMapper;

    @Value("${security.rate-limit.per-account-per-minute:100}")
    private int perAccount;

    @Value("${security.rate-limit.per-ip-per-minute:300}")
    private int perAddress;

    @Override
    protected boolean shouldNotFilter(HttpServletRequest request) {
        String path = request.getServletPath();
        // Container probes and the API docs are not worth counting.
        return path.startsWith("/actuator")
                || path.startsWith("/swagger-ui")
                || path.startsWith("/v3/api-docs");
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response,
                                    FilterChain chain) throws ServletException, IOException {
        String ip = ClientIp.of(request);
        String account = accountOf(request);

        // Address first: an unauthenticated flood has no account to charge.
        Long addressCount = hit(String.format(ADDRESS_KEY, ip));
        if (exceeded(addressCount, perAddress)) {
            reject(response, perAddress, "Quá nhiều yêu cầu từ địa chỉ này. Vui lòng thử lại sau một phút.");
            return;
        }

        Long accountCount = null;
        if (account != null) {
            accountCount = hit(String.format(ACCOUNT_KEY, account));
            if (exceeded(accountCount, perAccount)) {
                reject(response, perAccount, "Quá nhiều yêu cầu từ tài khoản này. Vui lòng thử lại sau một phút.");
                return;
            }
        }

        // Report whichever budget is tighter, so a client tuning its pace sees
        // the limit that will actually stop it.
        if (accountCount != null && perAccount - accountCount < perAddress - addressCount) {
            describe(response, perAccount, accountCount);
        } else {
            describe(response, perAddress, addressCount);
        }

        chain.doFilter(request, response);
    }

    /** Subject of a valid bearer token, or null for anonymous traffic. */
    private String accountOf(HttpServletRequest request) {
        String header = request.getHeader("Authorization");
        if (header == null || !header.startsWith("Bearer ")) {
            return null;
        }
        try {
            String token = header.substring(7);
            return jwtService.validateToken(token) ? jwtService.extractEmail(token) : null;
        } catch (Exception e) {
            // An unreadable token is anonymous traffic; the address limit still
            // applies and JwtAuthFilter will reject it in a moment anyway.
            return null;
        }
    }

    /** Count this request, or null when Redis will not answer. */
    private Long hit(String key) {
        try {
            Long count = redis.opsForValue().increment(key);
            if (count != null && count == 1L) {
                redis.expire(key, WINDOW);
            }
            return count;
        } catch (Exception e) {
            log.warn("Rate limit skipped for {}: {}", key, e.getMessage());
            return null;
        }
    }

    private boolean exceeded(Long count, int limit) {
        return count != null && count > limit;
    }

    private void describe(HttpServletResponse response, int limit, Long count) {
        if (count == null) return;
        response.setHeader("X-RateLimit-Limit", String.valueOf(limit));
        response.setHeader("X-RateLimit-Remaining", String.valueOf(Math.max(0, limit - count)));
        response.setHeader("X-RateLimit-Reset",
                String.valueOf(Instant.now().plus(WINDOW).getEpochSecond()));
    }

    private void reject(HttpServletResponse response, int limit, String message) throws IOException {
        response.setStatus(HttpStatus.TOO_MANY_REQUESTS.value());
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        response.setCharacterEncoding("UTF-8");
        response.setHeader("Retry-After", String.valueOf(WINDOW.toSeconds()));
        response.setHeader("X-RateLimit-Limit", String.valueOf(limit));
        response.setHeader("X-RateLimit-Remaining", "0");
        objectMapper.writeValue(response.getWriter(), new GlobalExceptionHandler.ErrorResponse(
                HttpStatus.TOO_MANY_REQUESTS.value(),
                HttpStatus.TOO_MANY_REQUESTS.getReasonPhrase(),
                message));
    }
}
