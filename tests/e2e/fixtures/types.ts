import { StartedPostgreSqlContainer } from "@testcontainers/postgresql";
import { StartedTestContainer, StartedNetwork } from "testcontainers";

declare global {
  var pgContainer: StartedPostgreSqlContainer | undefined;
  var payContainer: StartedTestContainer | undefined;
  var sharedNetwork: StartedNetwork | undefined;
}

export { };
