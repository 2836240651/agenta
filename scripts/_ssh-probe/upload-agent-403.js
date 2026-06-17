const { Client } = require('ssh2');
const fs = require('fs');
const path = require('path');

const zipLocal = path.resolve(
  __dirname,
  '../../commander-agent-t260220-main/commander-agent-t260220-main/build/bin/Agent_windows_amd64_4.0.3.zip'
);
const yamlLocal = path.resolve(
  __dirname,
  '../../commander-server-t260220-main/commander-server-t260220-main/etc/config/version.yaml'
);
const remoteDir = '/opt/1panel/www/sites/www.yoto.work/index/downloads';
const remoteZip = `${remoteDir}/Agent_windows_amd64_4.0.3.zip`;
const remoteYaml = '/data/commander/config/version.yaml';

if (!fs.existsSync(zipLocal)) {
  console.error('missing zip:', zipLocal);
  process.exit(1);
}
if (!fs.existsSync(yamlLocal)) {
  console.error('missing yaml:', yamlLocal);
  process.exit(1);
}

const conn = new Client();
conn.on('ready', () => {
  conn.sftp((err, sftp) => {
    if (err) { console.error(err); process.exit(1); }
    conn.exec(`mkdir -p ${remoteDir}`, () => {
      console.log('uploading zip ->', remoteZip);
      sftp.fastPut(zipLocal, remoteZip, (e1) => {
        if (e1) { console.error('zip upload failed', e1.message); process.exit(1); }
        console.log('uploaded zip');
        console.log('uploading yaml ->', remoteYaml);
        sftp.fastPut(yamlLocal, remoteYaml, (e2) => {
          if (e2) { console.error('yaml upload failed', e2.message); process.exit(1); }
          console.log('uploaded yaml');
          const verify = [
            `chmod 644 ${remoteZip} ${remoteYaml}`,
            `ls -la ${remoteZip}`,
            'echo === ZIP HEAD ===',
            'curl -sI https://www.yoto.work/downloads/Agent_windows_amd64_4.0.3.zip | head -8',
            'echo === AGENT VERSION API ===',
            'curl -s http://127.0.0.1:34206/api/v1/openapi/agent_version | head -c 500',
            'echo',
          ].join('; ');
          conn.exec(verify, (e3, stream) => {
            stream.on('data', d => process.stdout.write(d));
            stream.stderr.on('data', d => process.stderr.write(d));
            stream.on('close', (code) => { conn.end(); process.exit(code || 0); });
          });
        });
      });
    });
  });
}).on('error', e => { console.error('SSH error:', e.message); process.exit(1); })
  .connect({ host: '124.223.27.98', port: 22, username: 'root', password: 'Hyh3202276686@@@', readyTimeout: 120000 });
