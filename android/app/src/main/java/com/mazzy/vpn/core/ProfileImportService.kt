package com.mazzy.vpn.core

import java.nio.charset.StandardCharsets

interface ProfileDocumentStore {
    fun read(profileId: String): String?
    fun replaceAtomically(profileId: String, document: String)
    fun restore(profileId: String, document: String?)
    /** Durable write-ahead marker used to recover if the process dies between stores. */
    fun writeImportJournal(profileId: String, journal: String) {}
    fun readImportJournal(profileId: String): String? = null
    fun clearImportJournal(profileId: String) {}
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
        recoverInterruptedImport(profile.profileId)
        val previousDocument = documents.read(profile.profileId)
        val previousSecrets = profile.credentials.keys.associateWith { secrets.read(secretKey(profile.profileId, it)) }
        val journal = org.json.JSONObject().apply {
            put("document", previousDocument ?: org.json.JSONObject.NULL)
            put("secrets", org.json.JSONObject().apply {
                previousSecrets.forEach { (name, value) -> put(name, value ?: org.json.JSONObject.NULL) }
            })
        }.toString()
        documents.writeImportJournal(profile.profileId, journal)
        try {
            profile.credentials.forEach { (name, value) -> secrets.write(secretKey(profile.profileId, name), value) }
            documents.replaceAtomically(profile.profileId, jsonWithoutCredentials(json))
            documents.clearImportJournal(profile.profileId)
            return profile
        } catch (failure: Throwable) {
            runCatching { documents.restore(profile.profileId, previousDocument) }
            previousSecrets.forEach { (name, value) ->
                runCatching {
                    if (value == null) secrets.delete(secretKey(profile.profileId, name))
                    else secrets.write(secretKey(profile.profileId, name), value)
                }
            }
            documents.clearImportJournal(profile.profileId)
            throw failure
        }
    }

    private fun jsonWithoutCredentials(json: String): String {
        val root = org.json.JSONObject(json)
        root.remove("credentials")
        root.put("credentials", org.json.JSONObject())
        return root.toString()
    }

    private fun recoverInterruptedImport(profileId: String) {
        val raw = documents.readImportJournal(profileId) ?: return
        runCatching {
            val journal = org.json.JSONObject(raw)
            val previous = journal.opt("document")
            documents.restore(profileId, if (previous == org.json.JSONObject.NULL) null else previous as String)
            val values = journal.optJSONObject("secrets") ?: org.json.JSONObject()
            values.keys().forEach { name ->
                val value = values.opt(name)
                if (value == org.json.JSONObject.NULL) secrets.delete(secretKey(profileId, name))
                else secrets.write(secretKey(profileId, name), value as String)
            }
            documents.clearImportJournal(profileId)
        }.getOrElse { throw IllegalStateException("profile-import-recovery-failed", it) }
    }

    private fun secretKey(profileId: String, name: String) = "profile:$profileId:$name"
}
