package dev.chickinfry.chik

import java.io.File

// Resolves the chik binary. An explicit setting wins; otherwise the first chik
// on PATH is used. Resolving here turns a missing binary into one message
// instead of a process-start failure with a stack trace.
internal object ChikServer {
    private val binaryName =
        if (System.getProperty("os.name").startsWith("Windows", ignoreCase = true)) "chik.exe" else "chik"

    fun resolve(configured: String, searchPath: String = System.getenv("PATH").orEmpty()): String? {
        if (configured.isNotEmpty()) {
            val file = File(configured)
            return if (file.isAbsolute && !file.isFile) null else configured
        }
        return searchPath.split(File.pathSeparator)
            .asSequence()
            .filter { it.isNotEmpty() }
            .map { File(it, binaryName) }
            .firstOrNull { it.isFile }
            ?.path
    }
}

