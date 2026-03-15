
create mcp

```js
fetch('/api/m/mcp/servers', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    name: 'webpawm',
    transType: 'streamable', 
    url: 'http://localhost:8087/mcp',
    remark: '用于搜索'
  })
})
.then(r => r.json())
.then(console.log);


fetch('/api/m/mcp/servers/fi-54ionou2hq0w/activate', {
  method: 'PUT'
})
.then(r => r.json())
.then(console.log);

```
