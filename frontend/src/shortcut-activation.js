// A failed consent request is retried only after an explicit user action.
export function createShortcutActivation() {
 let failed=false,pending=null,generation=0;
 return {
  retry(){failed=false;},
  invalidate(){generation++;},
  fail(){failed=true;generation++;},
  async activate(start,eligible=()=>true){
   const requested=generation;
   if(failed||!eligible())return false;
   if(pending){
    // A reconnect waits for the old consent dialog instead of opening another.
    await pending;
    if(failed||requested!==generation||!eligible())return false;
   }
   if(pending)return false;
   const attempt=Promise.resolve().then(start);
   pending=attempt;
   try{await attempt;return !failed&&requested===generation&&eligible();}
   catch(error){failed=true;throw error;}
   finally{if(pending===attempt)pending=null;}
  }
 };
}
