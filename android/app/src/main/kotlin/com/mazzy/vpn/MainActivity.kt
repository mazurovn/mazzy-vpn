package com.mazzy.vpn

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import android.os.Bundle
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.content.ContextCompat
import com.mazzy.vpn.core.AndroidKeystoreSecretStore
import com.mazzy.vpn.core.ProfileImportService
import com.mazzy.vpn.core.SharedPreferencesDocumentStore

class MainActivity : Activity() {
    private lateinit var status: TextView

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        status = TextView(this).apply { text = getString(R.string.vpn_status_idle) }
        val start = Button(this).apply {
            text = getString(R.string.start_vpn)
            setOnClickListener { requestPermissionAndStart() }
        }
        val stop = Button(this).apply {
            text = getString(R.string.stop_vpn)
            setOnClickListener { startService(Intent(this@MainActivity, MazzyVpnService::class.java).setAction(MazzyVpnService.ACTION_STOP)) }
        }
        val import = Button(this).apply {
            text = getString(R.string.import_profile)
            setOnClickListener { startActivityForResult(Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                type = "application/json"
                addCategory(Intent.CATEGORY_OPENABLE)
            }, REQUEST_PROFILE_IMPORT) }
        }
        setContentView(LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(32, 32, 32, 32)
            addView(status)
            addView(import)
            addView(start)
            addView(stop)
        })
    }

    private fun requestPermissionAndStart() {
        val intent = VpnService.prepare(this)
        if (intent != null) {
            startActivityForResult(intent, REQUEST_VPN_PERMISSION)
        } else {
            startVpnService()
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == REQUEST_VPN_PERMISSION && resultCode == RESULT_OK) startVpnService()
        if (requestCode == REQUEST_PROFILE_IMPORT && resultCode == RESULT_OK) {
            val uri = data?.data ?: return
            val result = runCatching {
                contentResolver.openInputStream(uri)?.use { input ->
                    val bytes = input.readNBytes(256 * 1024 + 1)
                    require(bytes.size <= 256 * 1024) { "profile-too-large" }
                    ProfileImportService(
                        SharedPreferencesDocumentStore(this),
                        AndroidKeystoreSecretStore(this)
                    ).importProfile(bytes)
                } ?: error("profile-not-readable")
            }
            status.text = result.fold({ getString(R.string.profile_imported) }, { getString(R.string.profile_import_failed) })
        }
    }

    private fun startVpnService() {
        ContextCompat.startForegroundService(this, Intent(this, MazzyVpnService::class.java).setAction(MazzyVpnService.ACTION_START))
        status.text = getString(R.string.vpn_status_connecting)
    }

    companion object {
        private const val REQUEST_VPN_PERMISSION = 1001
        private const val REQUEST_PROFILE_IMPORT = 1002
    }
}
