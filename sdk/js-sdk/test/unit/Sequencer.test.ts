import { strict as assert } from 'assert';
import { Client } from '../../src';
// import { SequencerOptionsAutoIncrement } from '../../src/types/Sequencer';

describe('Sequencer.test.ts', () => {
  let client: Client;
  const storeName = 'etcd';

  beforeAll(() => {
    client = new Client();
  });

  it('should get next id success', async () => {
    let lastId = BigInt(0);
    let currentId = BigInt(0);
    const ids: bigint[] = [];
    for (let i = 0; i < 20; i++) {
      lastId = currentId;
      currentId = await client.sequencer.getNextId(storeName, 'user_info');
      assert(currentId > lastId);
      ids.push(currentId);
    }
    assert.equal(ids.length, 20);
    console.log('ids: %s', ids);
  });
});
