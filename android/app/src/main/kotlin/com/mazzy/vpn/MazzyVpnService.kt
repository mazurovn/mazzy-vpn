package com.mazzy.vpn

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Intent
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.VpnService
import android.os.Build
import android.os.IBinder
import android.content.pm.PackageManager
import androidx.core.app.NotificationCompat
import com.mazzy.vpn.core.AwgProfileRepository
import org.amnezia.awg.GoBackend
import java.util.concurrent.CancellationException
import java.util.concurrent.Executors
import java.util.concurrent.Future
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicLong

class MazzyVpnService : VpnService() {
    @Volatile private var state = State.IDLE
    private val executor = Executors.newSingleThreadExecutor { task ->
        Thread(task, "mazzy-awg-control")
    }
    private val dnsExecutor = Executors.newSingleThreadExecutor { task ->
        Thread(task, "mazzy-awg-dns")
    }
    @Volatile private var tunnelHandle = NO_HANDLE
    private val generation = AtomicLong(0)
    private val stateTransitionLock = Any()
    @Volatile private var lastStartId = 0
    @Volatile private var activeSessionTask: Future<*>? = null
    @Volatile private var destroyed = false
    @Volatile private var underlyingNetwork: Network? = null
    private val connectivity by lazy { getSystemService(CONNECTIVITY_SERVICE) as ConnectivityManager }
    private val networkCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) {
            underlyingNetwork = network
            scheduleUnderlyingNetworkUpdate(network)
        }

        override fun onLost(network: Network) {
            if (underlyingNetwork == network) {
                underlyingNetwork = null
                scheduleUnderlyingNetworkUpdate(null)
            }
        }
    }

    private fun scheduleUnderlyingNetworkUpdate(network: Network?) {
        val expectedGeneration = generation.get()
        runCatching {
            executor.execute {
                val mayUpdate = synchronized(stateTransitionLock) {
                    !destroyed && generation.get() == expectedGeneration &&
                        (state == State.CONNECTING || state == State.CONNECTED) &&
                        tunnelHandle != NO_HANDLE
                }
                if (mayUpdate) {
                    runCatching { setUnderlyingNetworks(network?.let { arrayOf(it) }) }
                        .onFailure { android.util.Log.w(TAG, "Underlying network update failed", it) }
                }
            }
        }.onFailure { android.util.Log.d(TAG, "Underlying network update ignored during teardown") }
    }

    override fun onCreate() {
        super.onCreate()
        destroyed = false
        connectivity.registerNetworkCallback(
            NetworkRequest.Builder()
                .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                .addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
                .build(),
            networkCallback
        )
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        lastStartId = startId
        when (intent?.action) {
            ACTION_START -> startSession(startId)
            ACTION_STOP -> stopSession(startId)
        }
        return START_NOT_STICKY
    }

    private fun startSession(startId: Int) {
        val sessionGeneration = synchronized(stateTransitionLock) {
            if (state != State.IDLE && state != State.ERROR) return
            state = State.PREPARING
            generation.incrementAndGet()
        }
        createChannel()
        startForeground(NOTIFICATION_ID, notification(getString(R.string.vpn_status_preparing)))

        val shouldConnect = synchronized(stateTransitionLock) {
            if (generation.get() != sessionGeneration || state != State.PREPARING) {
                false
            } else {
                state = State.CONNECTING
                true
            }
        }
        if (!shouldConnect) return
        updateNotification(getString(R.string.vpn_status_connecting))
        activeSessionTask = executor.submit {
            runCatching {
                val profile = AwgProfileRepository(this).load() ?: error("profile-not-imported")
                connectAwg(sessionGeneration, profile.displayName, profile.config)
            }.onFailure { failure ->
                android.util.Log.e(TAG, "Embedded AmneziaWG start failed", failure)
                disconnectAwg()
                failSessionIfCurrent(sessionGeneration, startId)
            }
        }
    }

    private fun stopSession(startId: Int, waitForTeardown: Boolean = false) {
        val stopGeneration = synchronized(stateTransitionLock) {
            if (state == State.IDLE || state == State.STOPPING) return
            val next = generation.incrementAndGet()
            state = State.STOPPING
            next
        }
        activeSessionTask?.cancel(true)
        updateNotification(getString(R.string.vpn_status_stopping))
        val cleanup: Future<*> = executor.submit {
            runCatching {
                disconnectAwg()
            }.onFailure { android.util.Log.e(TAG, "Embedded AmneziaWG stop failed", it) }
            synchronized(stateTransitionLock) {
                if (generation.get() == stopGeneration) {
                    state = State.IDLE
                    stopForeground(STOP_FOREGROUND_REMOVE)
                    stopSelfResult(startId)
                }
            }
        }
        if (waitForTeardown) {
            runCatching { cleanup.get(TEARDOWN_TIMEOUT_SECONDS, TimeUnit.SECONDS) }
                .onFailure {
                    android.util.Log.e(TAG, "Embedded AmneziaWG revoke teardown timed out", it)
                }
        }
    }

    private fun failSessionIfCurrent(sessionGeneration: Long, startId: Int) {
        synchronized(stateTransitionLock) {
            if (generation.get() != sessionGeneration) return
            publishState(State.ERROR)
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelfResult(startId)
        }
    }

    private fun publishState(newState: State) {
        state = newState
        val text = when (newState) {
            State.IDLE -> getString(R.string.vpn_status_idle)
            State.PREPARING -> getString(R.string.vpn_status_preparing)
            State.CONNECTING -> getString(R.string.vpn_status_connecting)
            State.CONNECTED -> getString(R.string.vpn_status_connected)
            State.STOPPING -> getString(R.string.vpn_status_stopping)
            State.ERROR -> getString(R.string.vpn_status_error)
        }
        updateNotification(text)
        sendBroadcast(Intent(ACTION_STATE_CHANGED).setPackage(packageName).putExtra(EXTRA_STATE, newState.name))
    }

    override fun onDestroy() {
        synchronized(stateTransitionLock) {
            destroyed = true
            generation.incrementAndGet()
        }
        runCatching { connectivity.unregisterNetworkCallback(networkCallback) }
        dnsExecutor.shutdownNow()
        activeSessionTask?.cancel(true)
        val cleanup = runCatching { executor.submit { disconnectAwg() } }.getOrNull()
        executor.shutdown()
        runCatching { cleanup?.get(TEARDOWN_TIMEOUT_SECONDS, TimeUnit.SECONDS) }
        if (!executor.isTerminated) executor.shutdownNow()
        stopForeground(STOP_FOREGROUND_REMOVE)
        state = State.IDLE
        super.onDestroy()
    }

    override fun onRevoke() {
        stopSession(lastStartId, waitForTeardown = true)
        super.onRevoke()
    }

    private fun connectAwg(sessionGeneration: Long, name: String, config: org.amnezia.awg.config.Config) {
        requireCurrentGeneration(sessionGeneration)
        require(VpnService.prepare(this) == null) { "vpn-not-authorized" }
        resolveEndpoints(config)
        requireCurrentGeneration(sessionGeneration)
        val builder = Builder().setSession(name).setBlocking(true)
        val awgInterface = config.getInterface()
        awgInterface.excludedApplications.forEach(builder::addDisallowedApplication)
        awgInterface.includedApplications.forEach(builder::addAllowedApplication)
        awgInterface.addresses.forEach { builder.addAddress(it.address, it.mask) }
        awgInterface.dnsServers.forEach { builder.addDnsServer(it.hostAddress!!) }
        awgInterface.dnsSearchDomains.forEach(builder::addSearchDomain)
        config.peers.flatMap { it.allowedIps }.forEach { builder.addRoute(it.address, it.mask) }
        builder.setMtu(awgInterface.mtu.orElse(1280))
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) builder.setMetered(false)
        setUnderlyingNetworks(underlyingNetwork?.let { arrayOf(it) })

        val tun = builder.establish() ?: error("tun-establish-failed")
        val handle = tun.use { GoBackend.awgTurnOn(name, it.detachFd(), config.toAwgUserspaceString()) }
        check(handle >= 0) { "awg-activation-failed-$handle" }
        tunnelHandle = handle
        requireCurrentGeneration(sessionGeneration)
        protectSocketIfPresent(GoBackend.awgGetSocketV4(handle), "ipv4")
        protectSocketIfPresent(GoBackend.awgGetSocketV6(handle), "ipv6")
        waitForHandshake(sessionGeneration, handle)
        requireCurrentGeneration(sessionGeneration)
        publishStateIfCurrent(sessionGeneration, State.CONNECTED)
    }

    private fun protectSocketIfPresent(socket: Int, family: String) {
        if (socket >= 0) check(protect(socket)) { "protect-$family-failed" }
    }

    private fun resolveEndpoints(config: org.amnezia.awg.config.Config) {
        repeat(DNS_RESOLUTION_ATTEMPTS) { attempt ->
            val resolution = dnsExecutor.submit<Boolean> {
                config.peers.all { peer ->
                    peer.endpoint.orElse(null)?.resolved?.orElse(null) != null
                }
            }
            val resolved = runCatching {
                resolution.get(DNS_RESOLUTION_TIMEOUT_SECONDS, TimeUnit.SECONDS)
            }.getOrDefault(false)
            if (resolved) return
            resolution.cancel(true)
            if (attempt + 1 < DNS_RESOLUTION_ATTEMPTS) Thread.sleep(DNS_RESOLUTION_RETRY_MILLIS)
        }
        error("awg-endpoint-dns-failed")
    }

    private fun waitForHandshake(sessionGeneration: Long, handle: Int) {
        val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(HANDSHAKE_TIMEOUT_SECONDS)
        while (System.nanoTime() < deadline) {
            requireCurrentGeneration(sessionGeneration)
            val runtime = GoBackend.awgGetConfig(handle).orEmpty()
            val handshake = runtime.lineSequence()
                .filter { it.startsWith("last_handshake_time_sec=") }
                .mapNotNull { it.substringAfter('=').toLongOrNull() }
                .maxOrNull() ?: 0L
            if (handshake > 0L) return
            Thread.sleep(HANDSHAKE_POLL_MILLIS)
        }
        error("awg-handshake-timeout")
    }

    private fun requireCurrentGeneration(sessionGeneration: Long) {
        if (generation.get() != sessionGeneration || Thread.currentThread().isInterrupted) {
            throw CancellationException("vpn-session-cancelled")
        }
    }

    private fun publishStateIfCurrent(sessionGeneration: Long, newState: State) {
        synchronized(stateTransitionLock) {
            requireCurrentGeneration(sessionGeneration)
            publishState(newState)
        }
    }

    private fun disconnectAwg() {
        val handle = tunnelHandle
        tunnelHandle = NO_HANDLE
        if (handle != NO_HANDLE) GoBackend.awgTurnOff(handle)
    }

    private fun updateNotification(text: String) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED
        ) {
            (getSystemService(NOTIFICATION_SERVICE) as NotificationManager)
                .notify(NOTIFICATION_ID, notification(text))
        }
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
        init {
            System.loadLibrary("wg-go")
        }

        const val ACTION_START = "com.mazzy.vpn.action.START"
        const val ACTION_STOP = "com.mazzy.vpn.action.STOP"
        const val ACTION_STATE_CHANGED = "com.mazzy.vpn.action.STATE_CHANGED"
        const val EXTRA_STATE = "state"
        private const val CHANNEL_ID = "vpn"
        private const val NOTIFICATION_ID = 1001
        private const val TAG = "MazzyVpnService"
        private const val NO_HANDLE = -1
        private const val HANDSHAKE_TIMEOUT_SECONDS = 35L
        private const val HANDSHAKE_POLL_MILLIS = 250L
        private const val DNS_RESOLUTION_ATTEMPTS = 3
        private const val DNS_RESOLUTION_RETRY_MILLIS = 1_000L
        private const val DNS_RESOLUTION_TIMEOUT_SECONDS = 2L
        private const val TEARDOWN_TIMEOUT_SECONDS = 5L
    }
}
