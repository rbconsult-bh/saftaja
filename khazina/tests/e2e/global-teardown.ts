import './fixtures/types';

async function globalTeardown() {
  console.log('🧹 Cleaning up...');

  if (globalThis.payContainer) {
    console.log('📋 Dumping final backend logs...');
    try {
      const stream = await globalThis.payContainer.logs();
      stream.on('data', (line: string) => {
        console.log(`[BACKEND FINAL]: ${line}`);
      });
      await new Promise(r => setTimeout(r, 2000));
    } catch {}
  }

  if (globalThis.tunnelProcess) {
    console.log('🔪 Killing tunnel process...');
    globalThis.tunnelProcess.kill();
  }

  await globalThis.payContainer?.stop();
  await globalThis.pgContainer?.stop();
  await globalThis.sharedNetwork?.stop();

  console.log('✅ Cleanup complete');
}

export default globalTeardown;
