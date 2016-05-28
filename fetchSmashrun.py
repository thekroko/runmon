from pyvirtualdisplay import Display
from selenium import webdriver

print("opening display")
display = Display(visible=0, size=(1024, 768))
display.start()

print("initializing browser")
browser = webdriver.Firefox()

print("fetching webpage")
browser.get('http://smashrun.com/mlinder314/list/')

print("scraping results")
runTable = browser.find_element_by_class_name('run-data').find_element_by_tag_name('tbody')
for track in runTable.find_elements_by_tag_name('tr'):
  print('reading track:')
  trackID = track.get_attribute('id')
  print('- ID = %s' % trackID)
  trackDate = track.find_element_by_class_name('date').text
  print('- Date: %s' % trackDate)
  trackDistance = track.find_element_by_class_name('distance').text
  print('- Distance: %s' % trackDistance)
  trackDuration = track.find_element_by_class_name('duration').text
  print('- Duration: %s' % trackDuration)
  with open("tracks.csv", "a") as myfile:
    myfile.write("%s,%s,%s,%s\n" % (trackID, trackDate, trackDistance, trackDuration))

print("shutting down")
browser.close()
display.stop()
