const { chromium } = require("/Users/erendt/code/lunar/frontend/node_modules/@playwright/test")

async function scrape(url, selector = "body") {
  const browser = await chromium.launch({ channel: "chrome", headless: true })
  const page = await browser.newPage()
  try {
    await page.goto(url, { waitUntil: "domcontentloaded", timeout: 30000 })
    await page.waitForTimeout(3000)
    const text = await page.innerText(selector)
    await browser.close()
    return text
  } catch (err) {
    await browser.close()
    return "Error scraping " + url + ": " + err.message
  }
}

async function main() {
  const url = process.argv[2]
  if (!url) {
    console.error("Please provide URL")
    process.exit(1)
  }
  const text = await scrape(url)
  console.log(text)
}

main()
