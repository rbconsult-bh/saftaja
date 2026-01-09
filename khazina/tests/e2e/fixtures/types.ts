import { StartedPostgreSqlContainer } from '@testcontainers/postgresql';
import { StartedTestContainer, StartedNetwork } from 'testcontainers';
import { ChildProcess } from 'child_process';

declare global {
  var pgContainer: StartedPostgreSqlContainer | undefined;
  var payContainer: StartedTestContainer | undefined;
  var sharedNetwork: StartedNetwork | undefined;
  var tunnelProcess: { kill: () => void } | undefined;
}

export { };
