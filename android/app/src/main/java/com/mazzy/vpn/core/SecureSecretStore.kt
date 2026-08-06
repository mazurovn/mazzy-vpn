package com.mazzy.vpn.core

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.nio.charset.StandardCharsets
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

interface SecureSecretStore {
    fun read(key: String): String?
    fun write(key: String, value: String)
    fun delete(key: String)
}

/** Keystore protects the encryption key; ciphertext is stored in private app preferences. */
class AndroidKeystoreSecretStore(context: Context) : SecureSecretStore {
    private val preferences = context.applicationContext.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
    private val key: SecretKey
        get() {
            val store = KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }
            val existing = store.getKey(KEY_ALIAS, null) as? SecretKey
            if (existing != null) return existing
            return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEYSTORE).apply {
                init(KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                    .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                    .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                    .setUserAuthenticationRequired(false)
                    .build())
            }.generateKey()
        }

    override fun read(key: String): String? {
        val encoded = preferences.getString(key, null) ?: return null
        return runCatching {
            val bytes = Base64.decode(encoded, Base64.NO_WRAP)
            val iv = bytes.copyOfRange(0, IV_LENGTH)
            Cipher.getInstance(TRANSFORMATION).run {
                init(Cipher.DECRYPT_MODE, this@AndroidKeystoreSecretStore.key, GCMParameterSpec(TAG_BITS, iv))
                String(doFinal(bytes.copyOfRange(IV_LENGTH, bytes.size)), StandardCharsets.UTF_8)
            }
        }.getOrNull()
    }

    override fun write(key: String, value: String) {
        val cipher = Cipher.getInstance(TRANSFORMATION).apply { init(Cipher.ENCRYPT_MODE, this@AndroidKeystoreSecretStore.key) }
        val payload = cipher.iv + cipher.doFinal(value.toByteArray(StandardCharsets.UTF_8))
        check(preferences.edit().putString(key, Base64.encodeToString(payload, Base64.NO_WRAP)).commit()) {
            "secret-store-write-failed"
        }
    }

    override fun delete(key: String) { preferences.edit().remove(key).commit() }

    private companion object {
        const val ANDROID_KEYSTORE = "AndroidKeyStore"
        const val KEY_ALIAS = "mazzy-vpn-profile-secrets-v1"
        const val PREFERENCES = "mazzy-vpn-secure-secrets-v1"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val IV_LENGTH = 12
        const val TAG_BITS = 128
    }
}
