import './fixtures/types';
import { PostgreSqlContainer } from "@testcontainers/postgresql";
import { GenericContainer, Network, Wait } from "testcontainers";
import path from 'path';
import { DB_CONFIG } from './fixtures/config';
import { seedDb } from './fixtures/db';

async function globalSetup() {
  console.log("🛜 Starting shared network...");
  const sharedNetwork = await new Network().start();

  console.log("🐘 Starting postgres...");
  const pgContainer = await new PostgreSqlContainer("postgres:18")
    .withNetwork(sharedNetwork)
    .withNetworkAliases('db')
    .withDatabase(DB_CONFIG.database)
    .withUsername(DB_CONFIG.user)
    .withPassword(DB_CONFIG.password)
    .start();

  const connectionString = pgContainer.getConnectionUri();
  process.env.DATABASE_URL = connectionString;

  console.log("🏗️ Building pay...");
  const builtPayContainer = await GenericContainer
    .fromDockerfile(path.resolve(__dirname, "../.."), "Dockerfile")
    .build();

  console.log("💰 Starting pay...");
  const payContainer = await builtPayContainer
    .withNetwork(sharedNetwork)
    .withEnvironment({
      DB_HOST: 'db',
      DB_PORT: '5432',
      DB_DATABASE: DB_CONFIG.database,
      DB_USER: DB_CONFIG.user,
      DB_PASSWORD: DB_CONFIG.password,
      PORT: '8080',
      MPGS_BASE_URL: process.env.MPGS_BASE_URL || '',
      MPGS_MERCHANT_ID: process.env.MPGS_MERCHANT_ID || '',
      MPGS_API_PASSWORD: process.env.MPGS_API_PASSWORD || '',
    })
    .withExposedPorts(8080)
    .withWaitStrategy(Wait.forHttp('/health', 8080))
    .start();

  console.log("🌱 Seeding database...");
  await seedDb();

  process.env.BASE_URL = `http://${payContainer.getHost()}:${payContainer.getMappedPort(8080)}`;

  globalThis.pgContainer = pgContainer;
  globalThis.payContainer = payContainer;
  globalThis.sharedNetwork = sharedNetwork;
}

export default globalSetup;
