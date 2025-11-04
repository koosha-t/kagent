import fs from 'fs'
import path from 'path'

export const size = {
  width: 378,
  height: 286,
}
export const contentType = 'image/png'

export default async function Icon() {
  // Read the kinagent.png file and return it directly
  const imagePath = path.join(process.cwd(), 'public', 'kinagent.png')
  const imageBuffer = fs.readFileSync(imagePath)

  return new Response(imageBuffer, {
    headers: {
      'Content-Type': 'image/png',
    },
  })
}
