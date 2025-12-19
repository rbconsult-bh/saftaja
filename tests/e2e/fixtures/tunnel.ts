import localtunnel from 'localtunnel';

export interface TunnelResult {
  process: { kill: () => void };
  publicUrl: string;
}

export async function startTunnel(localPort: number): Promise<TunnelResult> {
  console.log(`🚇 Starting Localtunnel for port ${localPort}...`);

  try {
    const tunnel = await localtunnel({ port: localPort });

    console.log(`✅ Localtunnel is live at: ${tunnel.url}`);

    return {
      process: {
        kill: () => tunnel.close(),
      },
      publicUrl: tunnel.url,
    };
  } catch (err) {
    throw new Error(`Failed to start localtunnel: ${err instanceof Error ? err.message : 'Unknown error'}`);
  }
}
