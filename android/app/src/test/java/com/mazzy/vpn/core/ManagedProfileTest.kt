package com.mazzy.vpn.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ManagedProfileTest {
    private val valid = """{"schema_version":1,"profile_id":"vless-test","display_name":"Test","protocol":"vless","endpoint":{"host":"vpn.example.com","port":443},"credentials":{"uuid":"123e4567-e89b-12d3-a456-426614174000"},"tls":{"enabled":true,"insecure":false,"server_name":"vpn.example.com"},"options":{},"dns":{},"routing":{}}"""

    @Test fun validatesAndRedactsCredentials() {
        val profile = ManagedProfileContractValidator.validate(valid).getOrThrow()
        assertEquals("vless", profile.protocol)
        assertTrue(profile.toString().contains("<redacted>"))
        assertTrue(!profile.toString().contains("123e4567"))
    }

    @Test(expected = IllegalArgumentException::class)
    fun rejectsUnknownRootKey() = ManagedProfileContractValidator.validate(valid.dropLast(1) + ",\"unexpected\":true}")

    @Test(expected = IllegalArgumentException::class)
    fun rejectsOversizedImport() = ManagedProfileContractValidator.validate(valid + " ".repeat(ManagedProfileContractValidator.MAX_IMPORT_BYTES))
}
