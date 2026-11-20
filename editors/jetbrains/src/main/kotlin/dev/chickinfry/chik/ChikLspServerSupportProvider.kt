package dev.chickinfry.chik

import com.intellij.execution.ExecutionException
import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.openapi.project.Project
import com.intellij.openapi.vfs.VirtualFile
import com.intellij.platform.lsp.api.LspServerSupportProvider
import com.intellij.platform.lsp.api.ProjectWideLspServerDescriptor

// One server per project. The server reads chik/*.chik across the project, so
// a project-wide descriptor matches how it resolves cross-file references.
//
// LspServerSupportProvider is deprecated since 2026.1.4 in favour of
// LspIntegrationProvider. It is used here because it is the entry point that
// exists in every 2026.1 build, and since-build is a branch, not a patch.
@Suppress("DEPRECATION")
internal class ChikLspServerSupportProvider : LspServerSupportProvider {
    override fun fileOpened(
        project: Project,
        file: VirtualFile,
        serverStarter: LspServerSupportProvider.LspServerStarter,
    ) {
        if (file.extension != "chik") return
        serverStarter.ensureServerStarted(ChikLspServerDescriptor(project))
    }
}

@Suppress("DEPRECATION")
private class ChikLspServerDescriptor(project: Project) :
    ProjectWideLspServerDescriptor(project, "Chik") {

    override fun isSupportedFile(file: VirtualFile): Boolean = file.extension == "chik"

    override fun createCommandLine(): GeneralCommandLine {
        val command = ChikServer.resolve(ChikSettings.getInstance().serverPath)
            ?: throw ExecutionException(
                "The chik language server was not found. Set its path in Settings | Tools | Chik.",
            )
        return GeneralCommandLine(command, "lsp", "--stdio")
    }
}

