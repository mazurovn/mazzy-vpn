package com.mazzy.vpn

import android.Manifest
import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.content.pm.PackageManager
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.content.ContextCompat
import com.mazzy.vpn.core.AwgProfileRepository
import java.io.ByteArrayOutputStream

class MainActivity : Activity() {
    private lateinit var status: TextView
    private var stateReceiverRegistered = false
    private val stateReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            val state = intent?.getStringExtra(MazzyVpnService.EXTRA_STATE) ?: return
            status.text = when (state) {
                "PREPARING" -> getString(R.string.vpn_status_preparing)
                "CONNECTING" -> getString(R.string.vpn_status_connecting)
                "CONNECTED" -> getString(R.string.vpn_status_connected)
                "STOPPING" -> getString(R.string.vpn_status_stopping)
                "ERROR" -> getString(R.string.vpn_status_error)
                else -> getString(R.string.vpn_status_idle)
            }
        }
    }

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
                type = "*/*"
                addCategory(Intent.CATEGORY_OPENABLE)
                putExtra(Intent.EXTRA_MIME_TYPES, arrayOf("text/plain", "application/x-wireguard-profile"))
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
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), REQUEST_NOTIFICATION_PERMISSION)
        }
    }

    override fun onStart() {
        super.onStart()
        if (!stateReceiverRegistered) {
            ContextCompat.registerReceiver(
                this,
                stateReceiver,
                IntentFilter(MazzyVpnService.ACTION_STATE_CHANGED),
                ContextCompat.RECEIVER_NOT_EXPORTED
            )
            stateReceiverRegistered = true
        }
    }

    override fun onStop() {
        if (stateReceiverRegistered) {
            unregisterReceiver(stateReceiver)
            stateReceiverRegistered = false
        }
        super.onStop()
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
        if (requestCode == REQUEST_VPN_PERMISSION) {
            if (resultCode == RESULT_OK) startVpnService()
            else status.text = getString(R.string.vpn_status_error)
        }
        if (requestCode == REQUEST_PROFILE_IMPORT && resultCode == RESULT_OK) {
            val uri = data?.data ?: return
            val result = runCatching {
                contentResolver.openInputStream(uri)?.use { input ->
                    val bytes = readBounded(input, 256 * 1024 + 1)
                    require(bytes.size <= 256 * 1024) { "profile-too-large" }
                    val name = uri.lastPathSegment?.substringAfterLast('/') ?: "mazzy-awg.conf"
                    AwgProfileRepository(this).importProfile(name, bytes)
                } ?: error("profile-not-readable")
            }
            status.text = result.fold({ getString(R.string.profile_imported) }, { getString(R.string.profile_import_failed) })
        }
    }

    private fun startVpnService() {
        ContextCompat.startForegroundService(this, Intent(this, MazzyVpnService::class.java).setAction(MazzyVpnService.ACTION_START))
        status.text = getString(R.string.vpn_status_connecting)
    }

    private fun readBounded(input: java.io.InputStream, limit: Int): ByteArray {
        val output = ByteArrayOutputStream(minOf(limit, 16 * 1024))
        val buffer = ByteArray(16 * 1024)
        while (output.size() < limit) {
            val count = input.read(buffer, 0, minOf(buffer.size, limit - output.size()))
            if (count < 0) break
            if (count == 0) continue
            output.write(buffer, 0, count)
        }
        return output.toByteArray()
    }

    companion object {
        private const val REQUEST_VPN_PERMISSION = 1001
        private const val REQUEST_PROFILE_IMPORT = 1002
        private const val REQUEST_NOTIFICATION_PERMISSION = 1003
    }
}
