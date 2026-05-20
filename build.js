const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

function incrementMinorVersion(versionStr) {
    const match = versionStr.match(/^(\d+)\.(\d+)(?:\.(\d+))?(.*)$/);
    if (!match) {
        throw new Error(`Invalid version format: ${versionStr}`);
    }
    const major = parseInt(match[1], 10);
    const minor = parseInt(match[2], 10);
    const patch = match[3] !== undefined ? parseInt(match[3], 10) : 0;
    const prerelease = match[4] || '';

    const nextMinor = minor + 1;
    const nextPatch = 0;

    return `${major}.${nextMinor}.${nextPatch}${prerelease}`;
}

try {
    // 1. Ensure dist directory exists
    const distDir = path.join(__dirname, 'dist');
    if (!fs.existsSync(distDir)) {
        fs.mkdirSync(distDir, { recursive: true });
    }

    // 2. Read package.json to get the current version
    const packageJsonPath = path.join(__dirname, 'package.json');
    const packageJsonContent = fs.readFileSync(packageJsonPath, 'utf8');
    const packageJson = JSON.parse(packageJsonContent);
    const currentVersion = packageJson.version || '0.1.0';

    // 3. Determine output file name
    const isWin = process.argv.includes('--win');
    const outputFilename = isWin ? 'singlefile-extractor.exe' : 'singlefile-extractor';
    const outputPath = path.join('dist', outputFilename);

    console.log(`Building singlefile-extractor version v${currentVersion}...`);

    // 4. Run go build with current version in ldflags
    const ldflags = `-X main.version=${currentVersion}`;
    const cmd = `go build -trimpath -ldflags "${ldflags}" -o "${outputPath}" ./cmd/singlefile-extractor`;
    
    console.log(`Executing: ${cmd}`);
    execSync(cmd, { stdio: 'inherit' });

    console.log(`Successfully built ${outputPath} with version v${currentVersion}`);

    // 5. Increment the version for the next build
    const nextVersion = incrementMinorVersion(currentVersion);
    packageJson.version = nextVersion;

    // Preserve the exact indentation (using 4 spaces as seen in package.json)
    const updatedPackageJsonContent = JSON.stringify(packageJson, null, 4) + '\n';
    fs.writeFileSync(packageJsonPath, updatedPackageJsonContent, 'utf8');

    console.log(`Incremented version in package.json from v${currentVersion} to v${nextVersion} for the next build.`);
} catch (error) {
    console.error('Build failed:', error.message);
    process.exit(1);
}
