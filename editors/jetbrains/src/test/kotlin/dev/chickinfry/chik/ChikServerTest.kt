package dev.chickinfry.chik

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.io.File

// The server lookup decides whether the plugin starts at all, and its failure
// mode is a process that never launches. It is plain file logic, so it is
// tested without an IDE fixture.
class ChikServerTest {
    @Test
    fun configuredPathWinsOverPath() {
        val dir = createDir()
        try {
            val configured = File(dir, "custom-chik")
            configured.writeText("")
            assertEquals(configured.path, ChikServer.resolve(configured.path, "/usr/bin"))
        } finally {
            dir.deleteRecursively()
        }
    }

    @Test
    fun missingAbsolutePathIsReportedAsMissing() {
        val missing = File("/nonexistent/chickinfry/chik").path
        assertNull(ChikServer.resolve(missing, "/usr/bin"))
    }

    @Test
    fun configuredPathThatIsNotAbsoluteIsPassedThrough() {
        assertEquals("chik-dev", ChikServer.resolve("chik-dev", "/usr/bin"))
    }

    @Test
    fun emptyConfigurationSearchesPathInOrder() {
        val first = createDir()
        val second = createDir()
        try {
            val binary = File(second, binaryName())
            binary.writeText("")
            val found = ChikServer.resolve("", first.path + File.pathSeparator + second.path)
            assertEquals(binary.path, found)
        } finally {
            first.deleteRecursively()
            second.deleteRecursively()
        }
    }

    @Test
    fun emptyPathYieldsNoCommand() {
        assertNull(ChikServer.resolve("", ""))
    }

    @Test
    fun directoriesNamedLikeTheBinaryAreIgnored() {
        val dir = createDir()
        try {
            File(dir, binaryName()).mkdirs()
            assertNull(ChikServer.resolve("", dir.path))
        } finally {
            dir.deleteRecursively()
        }
    }

    private fun createDir(): File = kotlin.io.path.createTempDirectory("chik-server-").toFile()

    private fun binaryName(): String =
        if (System.getProperty("os.name").startsWith("Windows", ignoreCase = true)) "chik.exe" else "chik"
}

