package main

const (
	CYPHER_CONSTRAINT_TWEET    = `CREATE CONSTRAINT ON (t:Tweet) ASSERT t.id IS UNIQUE;`
	CYPHER_CONSTRAINT_USERNAME = `CREATE CONSTRAINT ON (u:TwitterUser) ASSERT u.screen_name IS UNIQUE;`
	CYPHER_CONSTRAINT_USERID   = `CREATE CONSTRAINT ON (u:TwitterUser) ASSERT u.id IS UNIQUE;`
	CYPHER_CONSTRAINT_HASHTAG  = `CREATE CONSTRAINT ON (h:Hashtag) ASSERT h.name IS UNIQUE;`
	CYPHER_CONSTRAINT_LINK     = `CREATE CONSTRAINT ON (l:Link) ASSERT l.url IS UNIQUE;`
	CYPHER_CONSTRAINT_SOURCE   = `CREATE CONSTRAINT ON (s:Source) ASSERT s.name IS UNIQUE;`

	CYPHER_TWEETS_MAX_ID = `
		MATCH 
			(u:TwitterUser {screen_name:{screen_name}})-[:POSTS]->(t:Tweet)
		RETURN 
			max(t.id) AS max_id
		`

	CYPHER_MENTIONS_MAX_ID = `	
		MATCH
			(u:TwitterUser {screen_name:{screen_name}})<-[m:MENTIONS]-(t:Tweet)
		WHERE
			m.method="mention_search"
		RETURN
			max(t.id) AS max_id
		`
	CYPHER_NEED_GRAPH_UPDATE = `
		MATCH 
			(a:TwitterUser)		
		WHERE 
			EXISTS(a.graph_updated) AND a.following <> 0
		WITH 
			a, size((a)-[:FOLLOWS]->()) as following_count
		WHERE 
			a.following <> following_count
		RETURN 
			a.id as id
		ORDER BY 
			a.graph_updated
		LIMIT 100
		`

	CYPHER_REMOVE_FOLLOWS_REL = `
		MATCH (mainUser:TwitterUser {id:{id}})-[rel:FOLLOWS]->()
		DELETE rel		
		`
	CYPHER_MERGE_FOLLOWING_IDS = `		
		UNWIND {ids} AS id
        WITH id
		MATCH (mainUser:TwitterUser {id:{user_id}})
        MERGE (followedUser:TwitterUser {id:id})		
		ON CREATE SET 
			followedUser.followers = 1,
			followedUser.following = -1,
			followedUser.graph_updated = 0
        MERGE (mainUser)-[:FOLLOWS]->(followedUser)		
		SET mainUser.graph_updated = timestamp()
		`

	CYPHER_USERS_NEED_COMPLETION = `
		MATCH (n1:TwitterUser)
		WHERE NOT EXISTS(n1.created_at)	AND EXISTS(n1.id)	
		RETURN n1.id as id
		LIMIT 100
		`

	CYPHER_USERS_UPDATE_BY_NAME = `	
		UNWIND {users} AS u
        WITH u
        MERGE (user:TwitterUser {screen_name:u.screen_name})
		ON CREATE SET user.graph_updated = 0
        SET user.id = u.id_str,
			user.name = u.name,
			user.created_at = u.created_at,
			user.location = u.location,
            user.followers = u.followers_count,
            user.following = u.friends_count,
            user.statuses = u.statuses_count,
            user.url = u.url,
            user.profile_image_url = u.profile_image_url,
			user.last_updated = timestamp()
		`
	CYPHER_USERS_UPDATE_BY_ID = `	
		UNWIND 
			{users} AS u
        WITH 
			u
        MERGE 
			(user:TwitterUser {id:u.id_str})
		ON CREATE SET 
			user.graph_updated = 0
        SET 
			user.screen_name = u.screen_name,
			user.name = u.name,
			user.created_at = u.created_at,
			user.location = u.location,
            user.followers = u.followers_count,
            user.following = u.friends_count,
            user.statuses = u.statuses_count,
            user.url = u.url,
            user.profile_image_url = u.profile_image_url,
			user.last_updated = timestamp()
		`

	CYPHER_FOLLOWERS_IMPORT = `	
		UNWIND {users} AS u
        WITH u
        MERGE (user:TwitterUser {screen_name:u.screen_name})
		ON CREATE SET 
			user.graph_updated = 0
        SET 
			user.name = u.name,
			user.id = u.id_str,
			user.created_at = u.created_at,
    		user.location = u.location,
            user.followers = u.followers_count,
            user.following = u.friends_count,
            user.statuses = u.statuses_count,
            user.url = u.url,
            user.profile_image_url = u.profile_image_url,
			user.last_updated = timestamp()
        MERGE (mainUser:TwitterUser {screen_name:{screen_name}})
        MERGE (user)-[:FOLLOWS]->(mainUser)	
		`

	CYPHER_TWEETS_IMPORT = `
		UNWIND {tweets} AS t
	    WITH t
	    ORDER BY t.id
	    WITH t,
	         t.entities AS e,
	         t.user AS u,
	         t.retweeted_status AS retweet
	    MERGE (tweet:Tweet {id:t.id})
	    SET tweet.created_at = t.created_at,	        
			tweet.text = t.text,
			tweet.favorites = t.favorite_count,
			tweet.retweets = t.retweet_count			
	    MERGE (user:User {screen_name:u.screen_name})
	    SET user.name = u.name,
	        user.location = u.location,
	        user.followers = u.followers_count,
	        user.following = u.friends_count,
	        user.statuses = u.statuses_count,
			user.created_at = u.created_at,
			user.favourites = u.favourites_count,
			user.listed = u.listed_count,
	        user.profile_image_url = u.profile_image_url		
	    MERGE (user)-[:POSTS]->(tweet)
	    MERGE (source:Source {name:REPLACE(SPLIT(t.source, ">")[1], "</a", "")})
	    MERGE (tweet)-[:USING]->(source)
	    FOREACH (h IN e.hashtags |
	      MERGE (tag:Hashtag {name:LOWER(h.text)})
	      MERGE (tag)<-[:TAGS]-(tweet)
	    )
	    FOREACH (u IN e.urls |
	      MERGE (url:Link {url:u.expanded_url})
	      MERGE (tweet)-[:CONTAINS]->(url)
	    )
	    FOREACH (m IN e.user_mentions |
	      MERGE (mentioned:User {screen_name:m.screen_name})
	      ON CREATE SET mentioned.name = m.name
	      MERGE (tweet)-[mts:MENTIONS]->(mentioned)
          SET mts.method = {mention_type}
	    )
	    FOREACH (r IN [r IN [t.in_reply_to_status_id] WHERE r IS NOT NULL] |
	      MERGE (reply_tweet:Tweet {id:r})		  
	      MERGE (tweet)-[:REPLY_TO]->(reply_tweet)
		  SET tweet.is_reply = true,
		      reply_tweet.is_replied = true
	    )
	    FOREACH (retweet_id IN [x IN [retweet.id] WHERE x IS NOT NULL] |
	        MERGE (retweet_tweet:Tweet {id:retweet_id})
	        MERGE (tweet)-[:RETWEETS]->(retweet_tweet)
			SET tweet.is_retweet = true				
	    )
		`
)
