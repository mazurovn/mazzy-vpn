package com.mazzy.vpn

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Intent
import android.net.VpnService
import android.os.IBinder
import androidx.core.app.NotificationCompat
import com.mazzy.vpn.core.SharedPreferencesDocumentStore

class MazzyVpnService : VpnService() {
    private var state = State.IDLE

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START -> startSession()
            ACTION_STOP -> stopSession()
        }
        return START_NOT_STICKY
    }

    private fun startSession() {
        if (state != State.IDLE && state != State.ERROR) return
        state = State.PREPARING
        createChannel()
        startForeground(NOTIFICATION_ID, notification(getString(R.string.vpn_status_preparing)))

        if (!SharedPreferencesDocumentStore(this).hasAnyProfile()) {
            failSession()
            return
        }
        // A real engine must be integrated before the system tunnel API is called.
        // Keeping the service in ERROR prevents a false connected state or traffic leak.
        failSession()
    }

    private fun stopSession() {
        state = State.STOPPING
        stopForeground(STOP_FOREGROUND_REMOVE)
        state = State.IDLE
        stopSelf()
    }

    private fun failSession() {
        state = State.ERROR
        updateNotification(getString(R.string.vpn_status_error))
        stopForeground(STOP_FOREGROUND_DETACH)
        stopSelf()
    }

    override fun onDestroy() {
        state = State.STOPPING
        stopForeground(STOP_FOREGROUND_REMOVE)
        state = State.IDLE
        super.onDestroy()
    }

    private fun updateNotification(text: String) {
        (getSystemService(NOTIFICATION_SERVICE) as NotificationManager).notify(NOTIFICATION_ID, notification(text))
    }

    private fun notification(text: String): Notification = NotificationCompat.Builder(this, CHANNEL_ID)
        .setSmallIcon(R.drawable.ic_stat_vpn)
        .setContentTitle(getString(R.string.app_name))
        .setContentText(text)
        .setOngoing(state != State.ERROR)
        .build()

    private fun createChannel() {
        val manager = getSystemService(NOTIFICATION_SERVICE) as NotificationManager
        manager.createNotificationChannel(NotificationChannel(CHANNEL_ID, getString(R.string.vpn_notification), NotificationManager.IMPORTANCE_LOW))
    }

    override fun onBind(intent: Intent): IBinder? = super.onBind(intent)

    private enum class State { IDLE, PREPARING, CONNECTING, CONNECTED, STOPPING, ERROR }

    companion object {
        const val ACTION_START = "com.mazzy.vpn.action.START"
        const val ACTION_STOP = "com.mazzy.vpn.action.STOP"
        private const val CHANNEL_ID = "vpn"
        private const val NOTIFICATION_ID = 1001
    }
}
