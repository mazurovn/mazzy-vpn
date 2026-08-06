package com.mazzy.vpn.core

import java.util.Locale

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
        require(profile.protocol.lowercase(Locale.ROOT) in protocols) { "unsupported_protocol" }
        require(profile.endpoint.host.length <= 253 && profile.endpoint.host.isNotBlank()) { "invalid_endpoint_host" }
        require(profile.endpoint.host.matches(Regex("^[A-Za-z0-9._:-]+$")) && ".." !in profile.endpoint.host) {
            "invalid_endpoint_host"
        }
        require(profile.endpoint.port in 1..65535) { "invalid_endpoint_port" }
        require(profile.protocolSpecificCredentials()) { "invalid_credentials" }
        require(
            !profile.tls.containsKey("insecure") || profile.tls["insecure"] is Boolean &&
                profile.tls["insecure"] == false
        ) { "insecure_tls" }
        // Parse the root a second time only to enforce the schema's closed root object.
        val keys = org.json.JSONObject(json).keys().asSequence().toSet()
        require(keys == rootKeys) { "unknown_or_missing_root_key" }
        profile
    }

    private fun ManagedProfile.protocolSpecificCredentials(): Boolean = when (protocol.lowercase(Locale.ROOT)) {
        "vless" -> credentials.keys == setOf("uuid") && credentials["uuid"]!!.matches(
            Regex("^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$")
        )
        "tuic" -> credentials.keys == setOf("uuid", "password") && credentials["uuid"]!!.matches(
            Regex("^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$")
        )
        "mieru", "naive" -> credentials.keys == setOf("username", "password")
        else -> credentials.keys == setOf("password")
    }

    private fun isForbiddenDisplayChar(char: Char): Boolean =
        char.code in 0..31 || char.code == 127 || char.code == 173 ||
            char.code in 0x061C..0x061C || char.code in 0x200B..0x200F ||
            char.code in 0x2028..0x202E || char.code in 0x2060..0x2069 || char.code == 0xFEFF
}
