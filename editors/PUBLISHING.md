# Package and publish the Chik editor extensions

## Build locally

Run each command block from the repository root. Building the packages
requires no marketplace credentials.

With Node.js 24, build one VSIX for both Visual Studio Marketplace and Open VSX.

```sh
cd editors/vscode
npm ci
npm run package -- --out chik-0.0.1.vsix
```

The package script runs `vsce package` and compiles the extension first.
In VS Code, select **Extensions > … > Install from VSIX** to try the result.
See the [VS Code packaging guide](https://code.visualstudio.com/api/working-with-extensions/publishing-extension#packaging-extensions).

With JDK 21, build and check the JetBrains plugin.

```sh
cd editors/jetbrains
./gradlew buildPlugin verifyPluginStructure --no-daemon
```

The wrapper produces `build/distributions/chik.zip`. In a supported
IDE, select **Settings > Plugins > ⚙ > Install Plugin from Disk** to try it.
See the [JetBrains Gradle task reference](https://plugins.jetbrains.com/docs/intellij/tools-intellij-platform-gradle-plugin-tasks.html#buildPlugin).

## Visual Studio Marketplace

1. Sign in to [Manage Publishers & Extensions](https://marketplace.visualstudio.com/manage/publishers/).
   Create the `chickinfry` publisher or get access from its owner. The package
   identity is `chickinfry.chik`.
2. Select the publisher, choose **New extension > Visual Studio Code**, and
   upload `editors/vscode/chik-0.0.1.vsix`.

For automated releases, register the GitHub Actions workflow under the
publisher's **Trusted Publishing** settings. Give the matching job
`id-token: write` and run this from `editors/vscode` with the package built.

```sh
npx vsce publish --oidc --packagePath chik-0.0.1.vsix
```

See the [Visual Studio Marketplace publishing guide](https://code.visualstudio.com/api/working-with-extensions/publishing-extension).

## Open VSX

1. Sign in at [Open VSX](https://open-vsx.org/) with GitHub, link the
   matching Eclipse account, and accept the Publisher Agreement.
2. Create an Open VSX access token. Supply it in the `OVSX_PAT` environment
   variable from your secret store. If the `chickinfry` namespace does
   not exist, create it with `npx --yes ovsx@1.4.0 create-namespace chickinfry`.
   If it exists, request publishing access from its owner. Namespace
   verification is a separate ownership request.
3. From `editors/vscode`, verify access and publish the same VSIX.

```sh
npx --yes ovsx@1.4.0 verify-pat chickinfry
npx --yes ovsx@1.4.0 publish chik-0.0.1.vsix
```

For automated releases, a namespace owner can register a GitHub Actions
trusted publisher. With `id-token: write` and no `OVSX_PAT` in the job,
run `npx --yes ovsx@1.4.0 publish --trusted-publishing chik-0.0.1.vsix`
from `editors/vscode`.
See the [Open VSX publishing guide](https://github.com/eclipse-openvsx/openvsx/wiki/Publishing-Extensions)
and [Open VSX CLI reference](https://github.com/eclipse-openvsx/openvsx/blob/main/cli/README.md).

## JetBrains Marketplace

1. Sign in to [JetBrains Marketplace](https://plugins.jetbrains.com/) and
   select or create the Chickinfry vendor profile for `dev.chickinfry.chik`.
   Accept the Marketplace Developer Agreement if prompted.
2. Prepare a signing certificate and private key. Supply their contents
   through `CERTIFICATE_CHAIN` and `PRIVATE_KEY` from your secret store.
   If the key is encrypted, also supply `PRIVATE_KEY_PASSWORD`. From
   `editors/jetbrains`, run the signing task.

   ```sh
   ./gradlew signPlugin --no-daemon --no-configuration-cache
   ```

3. For the first release, select **Upload plugin** in JetBrains Marketplace
   and upload `build/distributions/chik-signed.zip`. Use **Apache License 2.0**
   for the listing, give `https://github.com/chickinfry/chik-lang` as the source
   URL, and submit it for review.
4. For later releases, create a Marketplace token under **My Tokens** and
   supply it as `PUBLISH_TOKEN` along with the signing credentials. From
   `editors/jetbrains`, publish the new version.

   ```sh
   ./gradlew publishPlugin --no-daemon --no-configuration-cache
   ```

See [first upload](https://plugins.jetbrains.com/docs/marketplace/uploading-a-new-plugin.html),
[Gradle publishing](https://plugins.jetbrains.com/docs/intellij/publishing-plugin.html),
and [plugin signing](https://plugins.jetbrains.com/docs/intellij/plugin-signing.html).

Before publishing an update, change the version in the corresponding VS Code
manifest and lockfile or JetBrains `gradle.properties`, update the extension's
changelog, and rebuild the package.
