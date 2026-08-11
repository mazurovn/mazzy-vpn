package com.mazzy.vpn.core

object ManagedProfileContractValidator {
    const val MAX_IMPORT_BYTES = 256 * 1024

    private val protocols = setOf(
        "vless", "hysteria2", "mieru", "naive", "tuic", "shadowsocks2022",
        "trojan", "anytls", "shadowtls"
    )
    private val rootKeys = setOf(
        "schema_version", "profile_id", "display_name", "protocol", "endpoint",
        "credentials", "tls", "options", "dns", "routing"
    )

    fun validate(json: String): Result<ManagedProfile> = runCatching {
        val bytes = json.toByteArray(Charsets.UTF_8)
        require(bytes.size <= MAX_IMPORT_BYTES) { "profile_too_large" }
        require(json.isNotBlank()) { "profile_empty" }
        val profile = ManagedProfile.parse(json)
        require(profile.schemaVersion == 1) { "unsupported_schema_version" }
        require(profile.profileId.matches(Regex("^[a-z][a-z0-9-]{2,63}$"))) { "invalid_profile_id" }
        require(profile.displayName.length <= 128 && profile.displayName.none(::isForbiddenDisplayChar)) {
            "invalid_display_name"
        }
        // Protocol identifiers are wire values, not display labels: the schema is lowercase.
        require(profile.protocol in protocols) { "unsupported_protocol" }
        require(profile.endpoint.host.length <= 253 && profile.endpoint.host.isNotBlank()) { "invalid_endpoint_host" }
        require(profile.endpoint.host.matches(Regex("^[A-Za-z0-9._:-]+$")) && ".." !in profile.endpoint.host) {
            "invalid_endpoint_host"
        }
        require(profile.endpoint.port in 1..65535) { "invalid_endpoint_port" }
        require(profile.protocolSpecificCredentials()) { "invalid_credentials" }
        validateTls(profile.tls)
        validateDns(profile.dns)
        validateRouting(profile.routing)
        // Parse the root a second time only to enforce the schema's closed root object.
        val keys = org.json.JSONObject(json).keys().asSequence().toSet()
        require(keys == rootKeys) { "unknown_or_missing_root_key" }
        profile
    }

    private fun ManagedProfile.protocolSpecificCredentials(): Boolean = when (protocol) {
        "vless" -> credentials.keys == setOf("uuid") && credentials["uuid"]!!.matches(
            Regex("^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$")
        )
        "tuic" -> credentials.keys == setOf("uuid", "password") && credentials["uuid"]!!.matches(
            Regex("^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$")
        )
        "mieru", "naive" -> credentials.keys == setOf("username", "password")
        else -> credentials.keys == setOf("password")
    }

    private fun validateTls(tls: Map<String, Any?>) {
        require(tls["insecure"] == false) { "insecure_tls" }
        require(tls["enabled"] is Boolean && tls["insecure"] == false && tls.containsKey("server_name")) {
            "invalid_tls"
        }
        if (tls["enabled"] == false) {
            require(tls.keys == setOf("enabled", "insecure", "server_name") && tls["server_name"] == "") {
                "invalid_disabled_tls"
            }
            return
        }
        val allowed = setOf("enabled", "insecure", "server_name", "alpn", "utls_fingerprint", "reality_public_key", "reality_short_id", "certificate_public_key_sha256")
        require(tls.keys.all { it in allowed } && (tls["server_name"] as? String)?.isNotBlank() == true) { "invalid_tls" }
        (tls["alpn"] as? List<*>)?.let { values ->
            require(values.size <= 4 && values.distinct().size == values.size && values.all { it is String && it.isNotEmpty() && it.length <= 32 }) { "invalid_tls_alpn" }
        }
    }

    private fun validateDns(dns: Map<String, Any?>) {
        if (dns.isEmpty()) return
        require(dns.keys == setOf("strategy", "servers")) { "invalid_dns" }
        require(dns["strategy"] in setOf("prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only")) { "invalid_dns_strategy" }
        val servers = dns["servers"] as? List<*> ?: error("invalid_dns_servers")
        require(servers.isNotEmpty() && servers.size <= 3) { "invalid_dns_servers" }
        servers.forEach { item ->
            val server = item as? Map<*, *> ?: error("invalid_dns_server")
            require(server.keys == setOf("server", "server_port", "server_name", "path")) { "invalid_dns_server" }
            require(server["server"] is String && (server["server"] as String).isNotBlank()) { "invalid_dns_server" }
            require(server["server_name"] is String && (server["server_name"] as String).isNotBlank()) { "invalid_dns_server" }
            require(server["server_port"] is Number && (server["server_port"] as Number).toInt() in 1..65535) { "invalid_dns_server" }
            require(server["path"] is String && Regex("^/[^?#]{0,255}$").matches(server["path"] as String)) { "invalid_dns_server" }
        }
    }

    private fun validateRouting(routing: Map<String, Any?>) {
        if (routing.isEmpty()) return
        require(routing.keys == setOf("mode", "allow_lan") && routing["mode"] == "full-tunnel" && routing["allow_lan"] is Boolean) {
            "invalid_routing"
        }
    }

    private fun isForbiddenDisplayChar(char: Char): Boolean =
        char.code in 0..31 || char.code == 127 || char.code == 173 ||
            char.code in 0x061C..0x061C || char.code in 0x200B..0x200F ||
            char.code in 0x2028..0x202E || char.code in 0x2060..0x2069 || char.code == 0xFEFF
}
