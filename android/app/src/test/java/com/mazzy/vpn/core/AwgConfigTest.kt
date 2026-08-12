package com.mazzy.vpn.core

import org.amnezia.awg.config.Config
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test
import java.io.ByteArrayInputStream

class AwgConfigTest {
    @Test
    fun parsesPinnedAwgSyntaxAndAcceptsFullTunnelPolicy() {
        val config = parse(validConfig())
        validateFullTunnelConfig(config)
        assertEquals(4, config.getInterface().junkPacketCount.orElseThrow())
        assertEquals(1, config.peers.size)
    }

    @Test
    fun parserAcceptsConfigAfterUtf8BomIsRemoved() {
        val text = decodeProfileText("\uFEFF${validConfig()}".toByteArray())
        validateFullTunnelConfig(parse(text))
    }

    @Test
    fun rejectsShellHooks() {
        assertThrows(Exception::class.java) {
            parse(validConfig().replace("DNS = 1.1.1.1", "DNS = 1.1.1.1\nPostUp = id"))
        }
    }

    @Test
    fun rejectsProfileWithoutDns() {
        val config = parse(validConfig().replace("DNS = 1.1.1.1\n", ""))
        assertThrows(IllegalArgumentException::class.java) { validateFullTunnelConfig(config) }
    }

    @Test
    fun rejectsSplitTunnelProfile() {
        val config = parse(validConfig().replace("0.0.0.0/0", "10.0.0.0/8"))
        assertThrows(IllegalArgumentException::class.java) { validateFullTunnelConfig(config) }
    }

    @Test
    fun rejectsProfileWithoutEndpoint() {
        val config = parse(validConfig().replace("Endpoint = 198.51.100.10:51820\n", ""))
        assertThrows(IllegalArgumentException::class.java) { validateFullTunnelConfig(config) }
    }

    @Test
    fun acceptsProfileWithoutKeepalive() {
        val config = parse(validConfig().replace("PersistentKeepalive = 25", ""))
        validateFullTunnelConfig(config)
    }

    @Test
    fun acceptsKeepaliveAboveMobileRecommendation() {
        val config = parse(validConfig().replace("PersistentKeepalive = 25", "PersistentKeepalive = 60"))
        validateFullTunnelConfig(config)
    }

    @Test
    fun rejectsInvalidKeepalive() {
        val config = parse(validConfig().replace("PersistentKeepalive = 25", "PersistentKeepalive = invalid"))
        assertThrows(IllegalArgumentException::class.java) { validateFullTunnelConfig(config) }
    }

    private fun parse(text: String): Config =
        ByteArrayInputStream(text.toByteArray()).use(Config::parse)

    private fun validConfig() = """
        [Interface]
        PrivateKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
        Address = 10.77.0.2/32
        DNS = 1.1.1.1
        Jc = 4
        Jmin = 40
        Jmax = 70
        S1 = 1
        S2 = 2
        H1 = 123
        H2 = 456
        H3 = 789
        H4 = 987

        [Peer]
        PublicKey = AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
        AllowedIPs = 0.0.0.0/0
        Endpoint = 198.51.100.10:51820
        PersistentKeepalive = 25
    """.trimIndent()
}
