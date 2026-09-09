package com.example.demo.utils;

import jakarta.servlet.http.HttpServletRequest;

import java.net.InetAddress;
import java.net.UnknownHostException;

/**
 * Works out who a request actually came from.
 *
 * <p>Every browser request reaches this service through traefik and then the
 * Next.js rewrite, so the socket address is always the proxy container and
 * useless for telling one user from another. The real address is in
 * X-Forwarded-For.
 *
 * <p>That header is set by the client, though, so trusting it blindly hands an
 * attacker two gifts: evade any per-IP limit by rotating the header, and pin
 * the blame on somebody else's address to get them locked out. It is only read
 * when the request reached us from inside our own network - which our proxies
 * are, and a caller hitting the published port directly is not.
 */
public final class ClientIp {

    private ClientIp() {}

    public static String of(HttpServletRequest request) {
        String peer = request.getRemoteAddr();
        if (!isInternal(peer)) {
            return peer;
        }
        String forwarded = request.getHeader("X-Forwarded-For");
        if (forwarded == null || forwarded.isBlank()) {
            return peer;
        }
        // Left-most entry is the original client; the rest are the proxy chain.
        String first = forwarded.split(",")[0].trim();
        return first.isEmpty() ? peer : first;
    }

    /** True for loopback and the RFC 1918 private ranges. */
    private static boolean isInternal(String ip) {
        if (ip == null || ip.isBlank()) return false;
        try {
            InetAddress address = InetAddress.getByName(ip);
            return address.isLoopbackAddress()
                    || address.isSiteLocalAddress()
                    || address.isLinkLocalAddress()
                    || address.isAnyLocalAddress();
        } catch (UnknownHostException e) {
            return false;
        }
    }
}
