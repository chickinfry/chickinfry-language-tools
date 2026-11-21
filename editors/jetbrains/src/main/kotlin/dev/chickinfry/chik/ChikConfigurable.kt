package dev.chickinfry.chik

import com.intellij.openapi.fileChooser.FileChooser
import com.intellij.openapi.fileChooser.FileChooserDescriptor
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.components.JBLabel
import com.intellij.util.ui.FormBuilder
import javax.swing.JComponent
import javax.swing.JPanel

// Settings | Tools | Chik. One field: the path to the chik binary.
class ChikConfigurable : Configurable {
    private var panel: JPanel? = null
    private val path = TextFieldWithBrowseButton()

    override fun getDisplayName(): String = "Chik"

    override fun createComponent(): JComponent {
        if (panel == null) {
            val descriptor = FileChooserDescriptor(true, false, false, false, false, false)
            path.addActionListener {
                FileChooser.chooseFile(descriptor, null, null) { file ->
                    path.text = file.path
                }
            }
            panel = FormBuilder.createFormBuilder()
                .addLabeledComponent(JBLabel("chik binary:"), path)
                .addComponentToRightColumn(JBLabel("Leave empty to use chik from PATH."))
                .addComponentFillVertically(JPanel(), 0)
                .panel
        }
        return panel!!
    }

    override fun isModified(): Boolean = path.text.trim() != ChikSettings.getInstance().serverPath

    override fun apply() {
        ChikSettings.getInstance().setServerPath(path.text)
    }

    override fun reset() {
        path.text = ChikSettings.getInstance().serverPath
    }

    override fun disposeUIResources() {
        panel = null
    }
}

