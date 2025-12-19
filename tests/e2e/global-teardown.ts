import './fixtures/types';

async function globalTeardown() {
  console.log('🧹 Cleaning up...');

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
