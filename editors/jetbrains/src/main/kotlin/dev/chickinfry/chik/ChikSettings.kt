package dev.chickinfry.chik

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage

// One setting: where the chik binary lives. Empty means "find chik on PATH",
// which is the common case for a project that already uses the CLI.
@Service(Service.Level.APP)
@State(name = "ChikSettings", storages = [Storage("chik.xml")])
class ChikSettings : PersistentStateComponent<ChikSettingsData> {
    private var current = ChikSettingsData()

    override fun getState(): ChikSettingsData = current

    override fun loadState(state: ChikSettingsData) {
        current = state
    }

    val serverPath: String
        get() = current.serverPath.trim()

    fun setServerPath(value: String) {
        current.serverPath = value.trim()
    }

    companion object {
        fun getInstance(): ChikSettings =
            ApplicationManager.getApplication().getService(ChikSettings::class.java)
    }
}

// A plain bean so the XML serializer can read and write it.
class ChikSettingsData(var serverPath: String = "")

