package com.mazzy.vpn.core

import android.content.Context

class SharedPreferencesDocumentStore(context: Context) : ProfileDocumentStore {
    private val preferences = context.applicationContext
        .getSharedPreferences("mazzy-profile-documents-v1", Context.MODE_PRIVATE)

    override fun read(profileId: String): String? = preferences.getString(profileId, null)

    fun hasAnyProfile(): Boolean = preferences.all.keys.isNotEmpty()

    override fun replaceAtomically(profileId: String, document: String) {
        check(preferences.edit().putString(profileId, document).commit()) { "profile-document-write-failed" }
    }

    override fun restore(profileId: String, document: String?) {
        val editor = preferences.edit()
        if (document == null) editor.remove(profileId) else editor.putString(profileId, document)
        check(editor.commit()) { "profile-document-restore-failed" }
    }

    override fun writeImportJournal(profileId: String, journal: String) {
        check(preferences.edit().putString("__journal__:$profileId", journal).commit())
    }

    override fun readImportJournal(profileId: String): String? =
        preferences.getString("__journal__:$profileId", null)

    override fun clearImportJournal(profileId: String) {
        check(preferences.edit().remove("__journal__:$profileId").commit())
    }
}
