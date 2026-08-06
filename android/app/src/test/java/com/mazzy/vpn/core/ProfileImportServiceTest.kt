package com.mazzy.vpn.core

import org.junit.Assert.assertEquals
import org.junit.Test

class ProfileImportServiceTest {
    private val document = """{"schema_version":1,"profile_id":"vless-test","display_name":"Test","protocol":"vless","endpoint":{"host":"vpn.example.com","port":443},"credentials":{"uuid":"123e4567-e89b-12d3-a456-426614174000"},"tls":{"enabled":true,"insecure":false,"server_name":"vpn.example.com"},"options":{},"dns":{},"routing":{}}"""

    @Test fun importsSecretsSeparately() {
        val docs = FakeDocuments()
        val secrets = FakeSecrets()
        ProfileImportService(docs, secrets).importProfile(document.toByteArray())
        assertEquals("{}", org.json.JSONObject(docs.current!!).getJSONObject("credentials").toString())
        assertEquals("123e4567-e89b-12d3-a456-426614174000", secrets.values.values.single())
    }

    @Test fun rollsBackDocumentAndSecretOnFailure() {
        val docs = FakeDocuments().apply { current = "old"; fail = true }
        val secrets = FakeSecrets().apply { values["profile:vless-test:uuid"] = "old-secret" }
        try { ProfileImportService(docs, secrets).importProfile(document.toByteArray()) }
        catch (_: IllegalStateException) { /* expected */ }
        assertEquals("old", docs.current)
        assertEquals("old-secret", secrets.values["profile:vless-test:uuid"])
    }

    private class FakeDocuments : ProfileDocumentStore {
        var current: String? = null
        var fail = false
        override fun read(profileId: String) = current
        override fun replaceAtomically(profileId: String, document: String) { if (fail) throw IllegalStateException("write failed"); current = document }
        override fun restore(profileId: String, document: String?) { current = document }
    }

    private class FakeSecrets : SecureSecretStore {
        val values = mutableMapOf<String, String>()
        override fun read(key: String) = values[key]
        override fun write(key: String, value: String) { values[key] = value }
        override fun delete(key: String) { values.remove(key) }
    }
}
