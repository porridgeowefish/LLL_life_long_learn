const fs = require('node:fs');
const path = require('node:path');
const zlib = require('node:zlib');

// Small dependency-free writers keep packaging identical on all build hosts.
// Unix executable permissions are encoded explicitly rather than inherited
// from the Windows filesystem used for cross-building.
function entries(root, prefix, executablePaths = ['learn-with-ai']) {
  const result = [];
  function walk(relative) {
    const absolute = path.join(root, relative);
    const info = fs.lstatSync(absolute);
    if (info.isSymbolicLink()) throw new Error('Archive links are forbidden');
    if (info.isDirectory()) {
      for (const name of fs.readdirSync(absolute).sort()) walk(relative ? `${relative}/${name}` : name);
    } else {
      if (!info.isFile()) throw new Error('Archive requires regular files');
      result.push({ name: `${prefix}/${relative}`, data: fs.readFileSync(absolute), mode: executablePaths.includes(relative) ? 0o755 : 0o644 });
    }
  }
  walk('');
  return result;
}

function tarGzip(root, prefix, output, executablePaths) {
  const chunks = [];
  function octal(header, offset, width, value) {
    const encoded = value.toString(8).padStart(width - 1, '0');
    if (encoded.length >= width) throw new Error('Tar value exceeds header limit');
    header.write(`${encoded}\0`, offset, width, 'ascii');
  }
  for (const file of entries(root, prefix, executablePaths)) {
    const header = Buffer.alloc(512);
    let name = file.name;
    if (Buffer.byteLength(name) > 100) {
      const split = name.lastIndexOf('/');
      const parent = name.slice(0, split);
      name = name.slice(split + 1);
      if (Buffer.byteLength(parent) > 155 || Buffer.byteLength(name) > 100) throw new Error('Tar path exceeds ustar limit');
      header.write(parent, 345, 155, 'utf8');
    }
    header.write(name, 0, 100, 'utf8');
    octal(header, 100, 8, file.mode);
    octal(header, 108, 8, 0);
    octal(header, 116, 8, 0);
    octal(header, 124, 12, file.data.length);
    octal(header, 136, 12, 0);
    header.fill(32, 148, 156);
    header.write('0', 156);
    header.write('ustar\0', 257, 6, 'ascii');
    header.write('00', 263, 2, 'ascii');
    const sum = [...header].reduce((a, b) => a + b, 0);
    header.write(`${sum.toString(8).padStart(6, '0')}\0 `, 148, 8, 'ascii');
    chunks.push(header, file.data, Buffer.alloc((512 - file.data.length % 512) % 512));
  }
  chunks.push(Buffer.alloc(1024));
  fs.writeFileSync(output, zlib.gzipSync(Buffer.concat(chunks), { level: 9 }));
}

const crcTable = Array.from({ length: 256 }, (_, index) => {
  let value = index;
  for (let bit = 0; bit < 8; bit++) value = value & 1 ? 0xedb88320 ^ value >>> 1 : value >>> 1;
  return value >>> 0;
});
function crc32(data) {
  let crc = 0xffffffff;
  for (const byte of data) crc = crcTable[(crc ^ byte) & 255] ^ crc >>> 8;
  return (crc ^ 0xffffffff) >>> 0;
}
function zip(root, prefix, output) {
  const local = [], central = [];
  let offset = 0;
  const files = entries(root, prefix);
  for (const file of files) {
    const name = Buffer.from(file.name, 'utf8');
    const data = zlib.deflateRawSync(file.data, { level: 9 });
    const checksum = crc32(file.data);
    const header = Buffer.alloc(30);
    header.writeUInt32LE(0x04034b50);
    header.writeUInt16LE(20, 4);
    header.writeUInt16LE(0x800, 6);
    header.writeUInt16LE(8, 8);
    header.writeUInt16LE(33, 12); // 1980-01-01
    header.writeUInt32LE(checksum, 14);
    header.writeUInt32LE(data.length, 18);
    header.writeUInt32LE(file.data.length, 22);
    header.writeUInt16LE(name.length, 26);
    local.push(header, name, data);
    const directory = Buffer.alloc(46);
    directory.writeUInt32LE(0x02014b50);
    directory.writeUInt16LE(0x314, 4);
    directory.writeUInt16LE(20, 6);
    directory.writeUInt16LE(0x800, 8);
    directory.writeUInt16LE(8, 10);
    directory.writeUInt16LE(33, 14);
    directory.writeUInt32LE(checksum, 16);
    directory.writeUInt32LE(data.length, 20);
    directory.writeUInt32LE(file.data.length, 24);
    directory.writeUInt16LE(name.length, 28);
    directory.writeUInt32LE(((0o100000 | file.mode) << 16) >>> 0, 38);
    directory.writeUInt32LE(offset, 42);
    central.push(directory, name);
    offset += header.length + name.length + data.length;
  }
  const directory = Buffer.concat(central);
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50);
  end.writeUInt16LE(files.length, 8);
  end.writeUInt16LE(files.length, 10);
  end.writeUInt32LE(directory.length, 12);
  end.writeUInt32LE(offset, 16);
  fs.writeFileSync(output, Buffer.concat([...local, directory, end]));
}
module.exports = { tarGzip, zip };
