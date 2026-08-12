package com.mazzy.vpn.core

import android.content.Context
import org.amnezia.awg.config.Config
import org.json.JSONObject
import java.io.ByteArrayInputStream
import java.nio.ByteBuffer
import java.nio.charset.CodingErrorAction
import java.nio.charset.StandardCharsets
import java.net.Inet4Address

/** Encrypted single-active-profile repository for the embedded AmneziaWG engine. */
class AwgProfileRepository(context: Context) {
    private val appContext = context.applicationContext
    private val secrets: SecureSecretStore = AndroidKeystoreSecretStore(appContext)
    fun importProfile(displayName: String, bytes: ByteArray): ImportedAwgProfile {
        require(bytes.isNotEmpty()) { "profile-empty" }
        require(bytes.size <= MAX_IMPORT_BYTES) { "profile-too-large" }
        require(bytes.none { it == 0.toByte() }) { "profile-binary" }
        val normalizedName = normalizeName(displayName)
        val text = decodeProfileText(bytes)
        val config = parse(text)
        validateFullTunnelConfig(config)

        val envelope = JSONObject()
            .put("schema", PROFILE_SCHEMA)
            .put("display_name", normalizedName)
            .put("config", text)
            .toString()
        secrets.write(PROFILE_SECRET, envelope)
        return ImportedAwgProfile(normalizedName, config)
    }

    fun load(): ImportedAwgProfile? {
        val envelope = secrets.read(PROFILE_SECRET) ?: return null
        val document = JSONObject(envelope)
        require(document.optString("schema") == PROFILE_SCHEMA) { "profile-schema-unsupported" }
        val name = document.getString("display_name")
        val text = document.getString("config")
        val config = parse(text)
        validateFullTunnelConfig(config)
        return ImportedAwgProfile(name, config)
    }

    fun hasProfile(): Boolean = secrets.read(PROFILE_SECRET) != null

    private fun parse(text: String): Config =
        ByteArrayInputStream(text.toByteArray(StandardCharsets.UTF_8)).use(Config::parse)

    private fun normalizeName(candidate: String): String {
        val stem = candidate.substringBeforeLast('.').trim()
            .replace(Regex("[^A-Za-z0-9_.=-]+"), "-")
            .trim('-')
            .take(15)
        return stem.ifBlank { "mazzy-awg" }
    }

    companion object {
        const val MAX_IMPORT_BYTES = 256 * 1024
        private const val PROFILE_SCHEMA = "mazzy-vpn.awg-profile.v1"
        private const val PROFILE_SECRET = "awg:active:profile"
    }
}

internal fun decodeProfileText(bytes: ByteArray): String = StandardCharsets.UTF_8.newDecoder()
    .onMalformedInput(CodingErrorAction.REPORT)
    .onUnmappableCharacter(CodingErrorAction.REPORT)
    .decode(ByteBuffer.wrap(bytes))
    .toString()
    .removePrefix("\uFEFF")

data class ImportedAwgProfile(val displayName: String, val config: Config)

internal fun validateFullTunnelConfig(config: Config) {
    require(config.peers.size == 1) { "profile-requires-one-peer" }
    val awgInterface = config.getInterface()
    require(awgInterface.addresses.isNotEmpty()) { "profile-has-no-address" }
    require(awgInterface.dnsServers.isNotEmpty()) { "profile-has-no-dns" }
    require(awgInterface.includedApplications.isEmpty() || awgInterface.excludedApplications.isEmpty()) {
        "profile-app-policy-conflict"
    }
    val hasIpv4Default = config.peers.single().allowedIps.any {
        it.mask == 0 && it.address is Inet4Address
    }
    require(hasIpv4Default) { "profile-has-no-ipv4-default-route" }
    val peer = config.peers.single()
    require(peer.endpoint.isPresent) { "profile-has-no-endpoint" }
    peer.persistentKeepalive.ifPresent { raw ->
        val keepalive = raw.toIntOrNull()
        require(keepalive != null && keepalive in 0..MAX_KEEPALIVE_SECONDS) {
            "profile-has-invalid-keepalive"
        }
    }
}

private const val MAX_KEEPALIVE_SECONDS = 65_535
