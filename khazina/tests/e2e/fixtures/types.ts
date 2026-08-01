import { StartedPostgreSqlContainer } from '@testcontainers/postgresql';
import { StartedTestContainer, StartedNetwork } from 'testcontainers';

declare global {
  var pgContainer: StartedPostgreSqlContainer | undefined;
  var payContainer: StartedTestContainer | undefined;
  var sharedNetwork: StartedNetwork | undefined;
  var tunnelProcess: { kill: () => void } | undefined;
  var testEncryptionKey: string | undefined;
  var testDatabaseUrl: string | undefined;
  var testBaseUrl: string | undefined;
}

export { };
