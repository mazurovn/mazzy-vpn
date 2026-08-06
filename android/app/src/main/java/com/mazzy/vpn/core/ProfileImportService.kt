package com.mazzy.vpn.core

import java.nio.charset.StandardCharsets

interface ProfileDocumentStore {
    fun read(profileId: String): String?
    fun replaceAtomically(profileId: String, document: String)
    fun restore(profileId: String, document: String?)
}

/** Imports a validated profile, moving credentials to secure storage and rolling back both stores on failure. */
class ProfileImportService(
    private val documents: ProfileDocumentStore,
    private val secrets: SecureSecretStore,
    private val maxBytes: Int = ManagedProfileContractValidator.MAX_IMPORT_BYTES
) {
    fun importProfile(bytes: ByteArray): ManagedProfile {
        require(bytes.size <= maxBytes) { "profile-too-large" }
        val json = bytes.toString(StandardCharsets.UTF_8)
        val profile = ManagedProfileContractValidator.validate(json).getOrThrow()
        val previousDocument = documents.read(profile.profileId)
        val previousSecrets = profile.credentials.keys.associateWith { secrets.read(secretKey(profile.profileId, it)) }
        try {
            profile.credentials.forEach { (name, value) -> secrets.write(secretKey(profile.profileId, name), value) }
            documents.replaceAtomically(profile.profileId, jsonWithoutCredentials(json))
            return profile
        } catch (failure: Throwable) {
            runCatching { documents.restore(profile.profileId, previousDocument) }
            previousSecrets.forEach { (name, value) ->
                runCatching {
                    if (value == null) secrets.delete(secretKey(profile.profileId, name))
                    else secrets.write(secretKey(profile.profileId, name), value)
                }
            }
            throw failure
        }
    }

    private fun jsonWithoutCredentials(json: String): String {
        val root = org.json.JSONObject(json)
        root.remove("credentials")
        root.put("credentials", org.json.JSONObject())
        return root.toString()
    }

    private fun secretKey(profileId: String, name: String) = "profile:$profileId:$name"
}
