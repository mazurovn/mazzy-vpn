package com.mazzy.vpn.core

import org.json.JSONObject

/** The closed managed-profile v1 document. Secret values are intentionally not exposed by toString(). */
data class ManagedProfile(
    val schemaVersion: Int,
    val profileId: String,
    val displayName: String,
    val protocol: String,
    val endpoint: Endpoint,
    val credentials: Map<String, String>,
    val tls: Map<String, Any?>,
    val options: Map<String, Any?>,
    val dns: Map<String, Any?>,
    val routing: Map<String, Any?>
) {
    override fun toString(): String =
        "ManagedProfile(schemaVersion=$schemaVersion, profileId='$profileId', " +
            "displayName='$displayName', protocol='$protocol', endpoint=$endpoint, " +
            "credentials=<redacted>, tls=${redacted(tls)}, options=${redacted(options)}, " +
            "dns=${redacted(dns)}, routing=${redacted(routing)})"

    fun withoutCredentials(): ManagedProfile = copy(credentials = emptyMap())

    private fun redacted(values: Map<String, Any?>): Map<String, Any?> = values.mapValues { (key, value) ->
        if (key.lowercase().contains("password") || key.lowercase().contains("secret") || key.lowercase().contains("token")) {
            "<redacted>"
        } else {
            value
        }
    }

    data class Endpoint(val host: String, val port: Int)

    companion object {
        fun parse(json: String): ManagedProfile {
            val root = JSONObject(json)
            val schemaVersion = root.requiredInt("schema_version")
            val profile = ManagedProfile(
                schemaVersion = schemaVersion,
                profileId = root.requiredString("profile_id"),
                displayName = root.requiredString("display_name"),
                protocol = root.requiredString("protocol"),
                endpoint = root.requiredObject("endpoint").let {
                    Endpoint(it.requiredString("host"), it.requiredInt("port"))
                },
                credentials = root.requiredObject("credentials").stringMap(),
                tls = root.requiredObject("tls").anyMap(),
                options = root.requiredObject("options").anyMap(),
                dns = root.requiredObject("dns").anyMap(),
                routing = root.requiredObject("routing").anyMap()
            )
            return profile
        }

        private fun JSONObject.requiredString(name: String): String =
            if (has(name) && !isNull(name) && get(name) is String) getString(name)
            else throw ProfileValidationException("missing-or-invalid-$name")

        private fun JSONObject.requiredInt(name: String): Int =
            if (has(name) && !isNull(name) && get(name) is Number) getInt(name)
            else throw ProfileValidationException("missing-or-invalid-$name")

        private fun JSONObject.requiredObject(name: String): JSONObject =
            if (has(name) && !isNull(name) && get(name) is JSONObject) getJSONObject(name)
            else throw ProfileValidationException("missing-or-invalid-$name")

        private fun JSONObject.stringMap(): Map<String, String> = keys().asSequence().associateWith {
            if (get(it) is String) getString(it) else throw ProfileValidationException("invalid-credential-$it")
        }

        private fun JSONObject.anyMap(): Map<String, Any?> = keys().asSequence().associateWith { key ->
            get(key).let { value -> if (value == JSONObject.NULL) null else value }
        }
    }
}

class ProfileValidationException(message: String) : IllegalArgumentException(message)
