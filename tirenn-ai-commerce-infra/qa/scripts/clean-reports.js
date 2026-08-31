import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const qaDir = path.resolve(__dirname, '..');
const frontendDir = path.resolve(__dirname, '../../frontend');

const targetDirs = [
  path.join(qaDir, 'playwright-report'),
  path.join(qaDir, 'test-results'),
  path.join(qaDir, 'blob-report'),
  path.join(frontendDir, 'playwright-report'),
  path.join(frontendDir, 'test-results'),
];

for (const dir of targetDirs) {
  try {
    if (fs.existsSync(dir)) {
      fs.rmSync(dir, { recursive: true, force: true });
      console.log(`[Clean] Removed ${dir}`);
    }
  } catch (err) {
    console.error(`[Clean] Error removing ${dir}:`, err.message);
  }
}
