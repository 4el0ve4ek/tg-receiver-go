# tg-receiver-go

###General Information
It's a composition of bots, which allow user to link two account in different social media and to resend memes between them.

For now, you can link vk and telegram accounts. You can send wall or text to vk bot and receive it in your telegram.

For this project was used MongoDB to store information about relations. Probably, should migrate to Neo4j.



###Thoughts:

It's need to be refactored, 'cause it seems like a mess.

I tried to make it less connected, so I have receiver interface, but code looks ugly a bit... And I don't know how to add another DAO object.

