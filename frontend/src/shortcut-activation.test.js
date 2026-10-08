import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createShortcutActivation} from './shortcut-activation.js';
test('reconnect does not repeat a failed consent request; explicit retry can',async()=>{
 const activation=createShortcutActivation();let calls=0;
 const start=async()=>{calls++;throw Error('Permission declined');};
 await assert.rejects(activation.activate(start));
 assert.equal(await activation.activate(start),false);
 assert.equal(calls,1);
 activation.retry();await assert.rejects(activation.activate(start));assert.equal(calls,2);
});
test('asynchronous native failure prevents reconnect activation',async()=>{
 const activation=createShortcutActivation();let calls=0;
 const start=async()=>{calls++;};
 assert.equal(await activation.activate(start),true);
 activation.fail();activation.invalidate();
 assert.equal(await activation.activate(start),false);assert.equal(calls,1);
});
test('reconnect waits for stale successful consent and starts a fresh listener',async()=>{
 const activation=createShortcutActivation();let resolve,calls=0;
 const old=activation.activate(()=>{calls++;return new Promise(r=>{resolve=r;});});
 await Promise.resolve();activation.invalidate();
 const fresh=activation.activate(async()=>{calls++;});
 assert.equal(calls,1);resolve();
 assert.equal(await old,false);assert.equal(await fresh,true);assert.equal(calls,2);
});
test('stale failed consent remains latched and is not retried by reconnect',async()=>{
 const activation=createShortcutActivation();let reject,calls=0;
 const old=activation.activate(()=>{calls++;return new Promise((_,r)=>{reject=r;});});
 await Promise.resolve();activation.invalidate();
 const fresh=activation.activate(async()=>{calls++;});
 reject(Error('declined'));
 await assert.rejects(old);await assert.rejects(fresh);assert.equal(calls,1);
 assert.equal(await activation.activate(async()=>{calls++;}),false);
});
