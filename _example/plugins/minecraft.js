
(function(){
	// const {EventEmitter} = require('node:events')
	const socket = require('socket')

	return {
		onload(data){
			console.info('loading minecraft plugin', JSON.stringify(data))
			var addr = data.addr
			{
				let i = addr.lastIndexOf(':')
				let local = addr.substring(0, i) + ':'
				this.mcaster = socket.dial('udp', '127.0.0.1:4445', local)
				addr = addr.substring(i + 1)
			}
			this.mcaster.on('error', (err)=>{
				if(err.toString().endsWith('write: connection refused')){
					return;
				}
				console.error('muticaster on error:', err)
			})
			console.debug('mcaster.local:', this.mcaster.local)
			var motd = data.name
			console.trace('motd:', motd, addr)
			this.cast_intv = setInterval(()=>{
				this.mcaster.send(`[MOTD]${motd}[/MOTD][AD]${addr}[/AD]`)
			}, 1500)
		},
		onunload(){
			console.info('unloading minecraft plugin')
			clearInterval(this.cast_intv)
			this.mcaster.close()
		}
	}
})()
