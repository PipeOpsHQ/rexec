const fs = require('fs');
const path = '../cmd/rexec/main.go';

try {
  let content = fs.readFileSync(path, 'utf8');
  
  const search = `                router.GET("/pricing", func(c *gin.Context) {
                        c.File(indexFile)
                })`;
                
  const replace = `                router.GET("/pricing", func(c *gin.Context) {
                        c.File(indexFile)
                })
                router.GET("/promo", func(c *gin.Context) {
                        c.File(indexFile)
                })`;

  if (content.includes(search)) {
    const newContent = content.replace(search, replace);
    fs.writeFileSync(path, newContent);
    console.log('Successfully patched main.go');
  } else {
    console.error('Could not find the target string in main.go');
    console.log('Content preview:', content.substring(content.indexOf('router.GET("/pricing"'), content.indexOf('router.GET("/pricing"') + 200));
    process.exit(1);
  }
} catch (e) {
  console.error('Error:', e);
  process.exit(1);
}
