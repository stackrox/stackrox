package util

import groovy.util.logging.Slf4j
import javax.mail.Folder
import javax.mail.Message
import javax.mail.MessagingException
import javax.mail.Session
import javax.mail.Store
import javax.mail.URLName
import javax.mail.search.SearchTerm

@Slf4j
class MailService {
    private Session session
    private Store store
    private Folder defaultFolder
    private Folder spamFolder
    private final String host
    private final String username
    private final String password
    private final URLName url
    private final String protocol = "imaps"
    private final String file = "INBOX"
    private boolean loggedIn = false

    MailService(String host, String username, String password) {
        this.host = host
        this.username = username
        this.password = password
        url = new URLName(protocol, host, 993, file, username, password)
    }

    void login() throws Exception {
        if (session == null) {
            session = Session.getInstance(new Properties(), null)
        }
        store = session.getStore(url)

        Timer t = new Timer(20, 3)
        Exception exception = null
        while (t.IsValid()) {
            try {
                store.connect()
                break
            } catch (Exception e) {
                log.debug "Connection to mail server failed... retrying."
                exception = e
            }
        }
        if (exception) {
            throw exception
        }
        defaultFolder = store.getFolder(url)
        defaultFolder.open(Folder.READ_WRITE)
        spamFolder = store.getFolder("[Gmail]/Spam")
        spamFolder.open(Folder.READ_WRITE)
        loggedIn = true
    }

    void logout() throws MessagingException {
        if (loggedIn) {
            try {
                defaultFolder.close(false)
                spamFolder.close(false)
                store.close()
                store = null
                session = null
            } catch (IllegalStateException ise) {
                log.warn("Error on logout - already logged out", ise)
            } catch (Exception e) {
                throw e
            }
            loggedIn = false
        }
    }

    void refreshConnection() throws MessagingException {
        logout()
        login()
    }

    Message[] searchMessages(SearchTerm term) throws Exception {
        try {
            refreshConnection() //refresh inbox contents
            return defaultFolder.search(term) + spamFolder.search(term)
        } catch (Exception e) {
            log.warn("could not search messages", e)
            throw e
        }
    }
}
