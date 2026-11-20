package dev.chickinfry.chik

import com.intellij.openapi.application.PathManager
import com.intellij.openapi.diagnostic.Logger
import org.jetbrains.plugins.textmate.api.TextMateBundleProvider
import org.jetbrains.plugins.textmate.api.TextMateBundleProvider.PluginBundle
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption

// Ships a VS Code-format TextMate bundle. The TextMate plugin reads that
// format natively, so VS Code and this plugin use one grammar file and cannot
// highlight the same code differently. The files are unpacked once into the
// IDE system directory because the provider hands out a real path.
internal class ChikTextMateBundleProvider : TextMateBundleProvider {
    override fun getBundles(): List<PluginBundle> {
        val directory = extract() ?: return emptyList()
        return listOf(PluginBundle("chik", directory))
    }

    private fun extract(): Path? {
        val target = Path.of(PathManager.getSystemPath(), "chik", "textmate")
        try {
            Files.createDirectories(target)
            for (name in BUNDLE_FILES) {
                val stream = javaClass.classLoader.getResourceAsStream("textmate/$name") ?: run {
                    LOG.warn("bundled grammar is missing: $name")
                    return null
                }
                stream.use { Files.copy(it, target.resolve(name), StandardCopyOption.REPLACE_EXISTING) }
            }
            return target
        } catch (error: Exception) {
            LOG.warn("could not unpack the Chik grammar", error)
            return null
        }
    }

    private companion object {
        val LOG: Logger = Logger.getInstance(ChikTextMateBundleProvider::class.java)
        val BUNDLE_FILES = listOf("package.json", "chik.tmLanguage.json", "language-configuration.json")
    }
}

