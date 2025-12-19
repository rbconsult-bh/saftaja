import './fixtures/types';
import path from 'path';
import { PostgreSqlContainer } from '@testcontainers/postgresql';
import { GenericContainer, Network, Wait } from 'testcontainers';
import { DB_CONFIG } from './fixtures/config';
import { startTunnel } from './fixtures/tunnel';
import { waitForHealthCheck } from './fixtures/health';

async function globalSetup() {
  console.log('🛜 Starting shared network...');
  const sharedNetwork = await new Network().start();

  console.log('🐘 Starting postgres...');
  const pgContainer = await new PostgreSqlContainer('postgres:18')
    .withNetwork(sharedNetwork)
    .withNetworkAliases('db')
    .withDatabase(DB_CONFIG.database)
    .withUsername(DB_CONFIG.user)
    .withPassword(DB_CONFIG.password)
    .start();

  process.env.DATABASE_URL = pgContainer.getConnectionUri();

  console.log('🏗️  Building pay...');
  const payImage = await GenericContainer
    .fromDockerfile(path.resolve(__dirname, '../..'), 'Dockerfile')
    .build();

  console.log('💰 Starting pay...');
  const payPort = 8080;
  const payContainer = await payImage
    .withNetwork(sharedNetwork)
    .withEnvironment({
      DB_HOST: 'db',
      DB_PORT: DB_CONFIG.port.toString(),
      DB_DATABASE: DB_CONFIG.database,
      DB_USER: DB_CONFIG.user,
      DB_PASSWORD: DB_CONFIG.password,
      PORT: payPort.toString(),
    })
    .withExposedPorts(payPort)
    .withWaitStrategy(Wait.forHttp('/health', payPort))
    .start();

  const localPort = payContainer.getMappedPort(payPort);

  console.log('🚇 Starting ephemeral tunnel...');
  const { process: tunnelProcess, publicUrl } = await startTunnel(localPort);

  console.log(`✅ App is live at: ${publicUrl}`);
  process.env.BASE_URL = publicUrl;

  await waitForHealthCheck(`${publicUrl}/health`, 15000);

  globalThis.pgContainer = pgContainer;
  globalThis.payContainer = payContainer;
  globalThis.sharedNetwork = sharedNetwork;
  globalThis.tunnelProcess = tunnelProcess;
}

export default globalSetup;

