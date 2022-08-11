const srcNextURL = window.location.href;
// https://www.mashibing.com/study?courseNo=284&sectionNo=36848&systemId=1
const nextIndex = srcNextURL.indexOf("sectionNo=");
const nextIndex2 = srcNextURL.indexOf("&", nextIndex);
let nextURL = srcNextURL.substring(0, nextIndex + 10) + "8848";
if (nextIndex2 > 0) {
    nextURL += srcNextURL.substring(nextIndex2);
}