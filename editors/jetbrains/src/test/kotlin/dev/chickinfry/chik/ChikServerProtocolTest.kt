package dev.chickinfry.chik

import com.google.gson.Gson
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Assume.assumeTrue
import org.junit.Test
import java.io.File
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit

// Starts the real language server and speaks LSP to it over stdio.
//
// The plugin's own tests only prove the binary was found. This proves the
// process answers the protocol, which is what the plugin depends on: a path
// that starts a process saying nothing would pass the lookup test and fail
// every user.
class ChikServerProtocolTest {
    private val server: String? = System.getenv("CHIK_SERVER")?.takeIf { File(it).isFile }

    @Test
    fun initializeAdvertisesTheFeaturesThePluginUses() {
        assumeTrue("set CHIK_SERVER to a built chik binary", server != null)
        val (process, replies) = start()
        try {
            send(process, mapOf("jsonrpc" to "2.0", "id" to 1, "method" to "initialize", "params" to emptyMap<String, Any>()))
            val answer = replies.poll(10, TimeUnit.SECONDS)
            assertNotNull("the server did not answer initialize", answer)
            val result = answer!!["result"] as Map<*, *>
            val capabilities = result["capabilities"] as Map<*, *>
            assertEquals(true, capabilities["definitionProvider"])
            assertEquals(true, capabilities["documentFormattingProvider"])
            assertNotNull(capabilities["completionProvider"])
        } finally {
            process.destroy()
        }
    }

    @Test
    fun aDocumentGetsTheParserDiagnostics() {
        assumeTrue("set CHIK_SERVER to a built chik binary", server != null)
        val (process, replies) = start()
        try {
            val uri = "file:///project/chik/api.chik"
            send(
                process,
                mapOf(
                    "jsonrpc" to "2.0",
                    "method" to "textDocument/didOpen",
                    "params" to mapOf(
                        "textDocument" to mapOf(
                            "uri" to uri,
                            "version" to 1,
                            "text" to "types {\n  Post = { author: Author }\n}\n",
                        ),
                    ),
                ),
            )
            val published = replies.poll(10, TimeUnit.SECONDS)
            assertNotNull("no diagnostics were published", published)
            assertEquals("textDocument/publishDiagnostics", published!!["method"])
            val params = published["params"] as Map<*, *>
            val diagnostics = params["diagnostics"] as List<*>
            assertTrue("the unknown type should be reported", diagnostics.isNotEmpty())
            val first = diagnostics.first() as Map<*, *>
            // The number is the parser's, so the IDE and chik check agree.
            assertEquals("CHK0301", first["code"])
        } finally {
            process.destroy()
        }
    }

    private fun start(): Pair<Process, LinkedBlockingQueue<Map<*, *>>> {
        val process = ProcessBuilder(server, "lsp", "--stdio").redirectErrorStream(false).start()
        val replies = LinkedBlockingQueue<Map<*, *>>()
        val reader = Thread {
            try {
                while (true) {
                    val message = readMessage(process) ?: break
                    replies.put(message)
                }
            } catch (_: Exception) {
                // The process ended; the queue simply stops filling.
            }
        }
        reader.isDaemon = true
        reader.start()
        return process to replies
    }

    private fun readMessage(process: Process): Map<*, *>? {
        var length = -1
        while (true) {
            val line = readLine(process) ?: return null
            if (line.isEmpty()) break
            val (name, value) = line.split(":", limit = 2).takeIf { it.size == 2 } ?: continue
            if (name.trim().equals("Content-Length", ignoreCase = true)) length = value.trim().toInt()
        }
        if (length < 0) return null
        val body = ByteArray(length)
        var read = 0
        while (read < length) {
            val count = process.inputStream.read(body, read, length - read)
            if (count < 0) return null
            read += count
        }
        @Suppress("UNCHECKED_CAST")
        return Gson().fromJson(body.toString(Charsets.UTF_8), Map::class.java) as Map<*, *>
    }

    private fun readLine(process: Process): String? {
        val bytes = StringBuilder()
        while (true) {
            val next = process.inputStream.read()
            if (next < 0) return null
            val char = next.toChar()
            if (char == '\n') return bytes.toString().trimEnd('\r')
            bytes.append(char)
        }
    }

    private fun send(process: Process, message: Map<String, Any?>) {
        val body = Gson().toJson(message)
        val header = "Content-Length: " + body.toByteArray(Charsets.UTF_8).size + "\r\n\r\n"
        process.outputStream.write(header.toByteArray(Charsets.US_ASCII))
        process.outputStream.write(body.toByteArray(Charsets.UTF_8))
        process.outputStream.flush()
    }
}
