import test from 'node:test';
import assert from 'node:assert/strict';
import {createAuthCoordinator} from './auth-state.js';
test('verification cannot overwrite logout or a newer login',async()=>{const auth=createAuthCoordinator();let finish;const pending=new Promise(resolve=>finish=resolve);let applied=false;const run=auth.run(()=>pending,()=>applied=true,()=>assert.fail('obsolete verification error'));auth.invalidate();finish({logged_in:true});await run;assert.equal(applied,false);});
test('obsolete verification errors do not sign out a newer login',async()=>{const auth=createAuthCoordinator();let finish;const pending=new Promise((_,reject)=>finish=reject);let failed=false;const run=auth.run(()=>pending,()=>{},()=>failed=true);auth.invalidate();finish(Error('old session'));await run;assert.equal(failed,false);});
