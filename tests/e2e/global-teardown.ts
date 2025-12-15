async function globalTeardown() {
  console.log("🧹 Cleaning up...");

  await globalThis.payContainer?.stop();
  await globalThis.pgContainer?.stop();
  await globalThis.sharedNetwork?.stop();
}

export default globalTeardown;
