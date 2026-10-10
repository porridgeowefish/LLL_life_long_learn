const fs = require("node:fs"),
  os = require("node:os"),
  path = require("node:path"),
  assert = require("node:assert/strict");
const { spawnSync } = require("node:child_process");
const { buildMsi } = require("./windows-msi");
const { repoRoot } = require("./package-layout");
function ps(script) {
  const r = spawnSync(
    "powershell.exe",
    [
      "-NoProfile",
      "-NonInteractive",
      "-EncodedCommand",
      Buffer.from(script, "utf16le").toString("base64"),
    ],
    { encoding: "utf8", windowsHide: true },
  );
  assert.equal(r.status, 0, r.stderr);
  return r.stdout.trim();
}
function install(msi, args, log) {
  const r = spawnSync(
    "msiexec.exe",
    [args[0], msi, ...args.slice(1), "/qn", "/norestart", "/l*v", log],
    { windowsHide: true },
  );
  assert([0, 3010].includes(r.status), `msiexec ${r.status}; inspect ${log}`);
}
function main(input) {
  if (process.platform !== "win32")
    throw new Error("MSI smoke requires Windows");
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), "learn-msi-smoke-"));
  const identity = `learn-with-ai-smoke-${process.pid}`;
  let installed;
  try {
    const metadata = ps(
      `$i=New-Object -ComObject WindowsInstaller.Installer; $d=$i.OpenDatabase('${path.resolve(input).replaceAll("'", "''")}',0); foreach($table in @('File','Shortcut','Dialog','Upgrade','Media')) {$v=$d.OpenView('SELECT * FROM '+$table);$v.Execute();$n=0;while($v.Fetch()){$n++};Write-Output ($table+':'+$n)}`,
    );
    assert.match(metadata, /Shortcut:2/);
    assert.match(metadata, /Dialog:/);
    assert.match(metadata, /Upgrade:/);
    assert.match(metadata, /Media:1/);
    const match = path
      .basename(input)
      .match(/learn-with-ai-(\d+\.\d+\.\d+)-windows-(amd64|arm64)\.msi/);
    assert(match, "Expected release MSI");
    const payload = path.join(
      path.dirname(path.resolve(input)),
      ".staging",
      `learn-with-ai-${match[1]}-windows-${match[2]}`,
    );
    assert(fs.existsSync(payload));
    const folder = path.join(scratch, "installed");
    const props = [`INSTALLFOLDER=${folder}`];
    const base = buildMsi(payload, scratch, "0.5.0", match[2], { identity })[0];
    const next = buildMsi(payload, scratch, "0.5.1", match[2], { identity })[0];
    install(base, ["/i", ...props], path.join(scratch, "install.log"));
    installed = base;
    assert(fs.existsSync(path.join(folder, "learn-with-ai.exe")));
    fs.writeFileSync(path.join(folder, "learner-note.md"), "preserve");
    const original = fs.readFileSync(path.join(folder, "README.md"));
    fs.unlinkSync(path.join(folder, "README.md"));
    install(base, ["/fa"], path.join(scratch, "repair.log"));
    assert.deepEqual(fs.readFileSync(path.join(folder, "README.md")), original);
    install(base, ["/i", ...props], path.join(scratch, "reinstall.log"));
    install(next, ["/i", ...props], path.join(scratch, "upgrade.log"));
    installed = next;
    const downgrade = spawnSync(
      "msiexec.exe",
      [
        "/i",
        base,
        "/qn",
        "/norestart",
        "/l*v",
        path.join(scratch, "downgrade.log"),
      ],
      { windowsHide: true },
    );
    assert.equal(downgrade.status, 1603);
    const smoke = spawnSync(
      process.execPath,
      [
        path.join(__dirname, "smoke.js"),
        path.join(folder, "learn-with-ai.exe"),
      ],
      { stdio: "inherit", windowsHide: true },
    );
    assert.equal(smoke.status, 0);
    const nativeResult = path.join(scratch, "desktop-smoke.json");
    const native = spawnSync(
      path.join(folder, "learn-with-ai.exe"),
      [
        "--workspace",
        path.join(scratch, "native-workspace"),
        "--desktop-smoke-result",
        nativeResult,
      ],
      { cwd: scratch, windowsHide: true, timeout: 60000, stdio: "inherit" },
    );
    assert.equal(native.status, 0, "Installed native desktop smoke failed");
    const evidence = JSON.parse(fs.readFileSync(nativeResult, "utf8"));
    assert.equal(evidence.ok, true);
    assert.equal(evidence.hasWindowIcon, true);
    const sourceImport = spawnSync(
      process.execPath,
      [
        path.join(__dirname, "source-import-smoke.js"),
        path.join(folder, "learn-with-ai.exe"),
      ],
      { windowsHide: true, stdio: "inherit", timeout: 120000 },
    );
    assert.equal(sourceImport.status, 0, "Installed parser lifecycle failed");
    install(next, ["/x"], path.join(scratch, "uninstall.log"));
    installed = null;
    assert(!fs.existsSync(path.join(folder, "learn-with-ai.exe")));
    assert.equal(
      fs.readFileSync(path.join(folder, "learner-note.md"), "utf8"),
      "preserve",
    );
    console.log(
      `MSI table inventory, isolated install/reinstall/repair/upgrade/downgrade/uninstall, native rendered window and icon, source import and data retention passed. Logs: ${scratch}`,
    );
  } finally {
    if (installed)
      spawnSync("msiexec.exe", ["/x", installed, "/qn", "/norestart"], {
        windowsHide: true,
      });
  }
}
module.exports = { main };
if (require.main === module) main(process.argv[2]);
