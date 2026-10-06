export function createVoiceSession({getSocket,clock=globalThis,notify=()=>{},uuid=()=>crypto.randomUUID()}) {
 let held=false,blocked=false,id=null,timer=null;
 const send=(state,sessionId,callback)=>{const socket=getSocket();if(socket?.connected)socket.emit('guildsync:voice-mute-hotkey',{state,sessionId},callback);};
 function stop(){blocked=true;if(timer!==null)clock.clearInterval(timer);timer=null;const old=id;id=null;if(old)send('released',old);}
 function onEdge(down){
  if(!down){held=false;stop();blocked=false;return;}
  if(held||blocked)return;held=true;
  if(!getSocket()?.connected){blocked=true;return;}
  const current=uuid();id=current;
  const ack=result=>{if(id!==current)return;if(!result?.ok){stop();notify(result?.message||'Voice mute is unavailable.')}else notify('Shortcut held — voice mute requested.');};
  send('pressed',current,ack);
  if(id===current)timer=clock.setInterval(()=>{if(id===current)send('heartbeat',current,result=>{if(id===current&&!result?.ok){stop();notify(result?.message||'Voice mute ended.');}});},2000);
 }
 return {onEdge,stop,onDisconnect(){if(timer!==null)clock.clearInterval(timer);timer=null;id=null;blocked=true;}};
}
