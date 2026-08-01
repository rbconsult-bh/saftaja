/**
 * Polls a health check endpoint until it returns OK or times out.
 */
export async function waitForHealthCheck(url: string, timeoutMs = 60000): Promise<void> {
  const start = Date.now();
  console.log(`⏱️  Waiting for: ${url}`);

  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url, {
        headers: {
          'bypass-tunnel-reminder': 'i guess we need this weird header :D',
          'User-Agent': 'curl/8.7.1',
        }
      });
      if (res.ok) {
        console.log(`✅ Health check passed (${res.status})`);
        return;
      }
    } catch {
      // ignore connection errors, keep polling
    }
    await new Promise((r) => setTimeout(r, 2000));
  }

  throw new Error(`❌ Timed out waiting for ${url}`);
}
